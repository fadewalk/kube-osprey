package notify

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/pkg/model"
)

type channelRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Endpoint string `json:"endpoint" binding:"required"`
	Secret   string `json:"secret"`
	Events   string `json:"events"`
	Enabled  *bool  `json:"enabled"`
}

var validChannelTypes = map[string]struct{}{
	"dingtalk": {},
	"wecom":    {},
	"feishu":   {},
	"slack":    {},
	"webhook":  {},
}

func sanitizeEvents(events string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n':
			return -1
		}
		return r
	}, events)
	if cleaned == "" {
		return model.NotifyEventPlanPending
	}
	return cleaned
}

// RegisterAdminRoutes wires the notification channel admin API onto the
// settings group (admin + audit middleware are applied by the caller).
func RegisterAdminRoutes(group *gin.RouterGroup) {
	g := group.Group("/notifications")
	g.GET("/channels", ListChannels)
	g.POST("/channels", CreateChannel)
	g.PUT("/channels/:id", UpdateChannel)
	g.DELETE("/channels/:id", DeleteChannel)
	g.POST("/channels/:id/test", TestChannel)
}

func ListChannels(c *gin.Context) {
	channels, err := model.ListNotificationChannels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

func (r channelRequest) normalize() (model.NotificationChannel, string) {
	channelType := strings.ToLower(r.Type)
	if _, ok := validChannelTypes[channelType]; !ok {
		return model.NotificationChannel{}, "unknown channel type"
	}
	if strings.TrimSpace(r.Endpoint) == "" {
		return model.NotificationChannel{}, "endpoint is required"
	}
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return model.NotificationChannel{
		Name:     strings.TrimSpace(r.Name),
		Type:     channelType,
		Endpoint: strings.TrimSpace(r.Endpoint),
		Events:   sanitizeEvents(r.Events),
		Enabled:  enabled,
	}, ""
}

func CreateChannel(c *gin.Context) {
	var req channelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	channel, problem := req.normalize()
	if problem != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": problem})
		return
	}
	if strings.TrimSpace(req.Secret) != "" {
		channel.Secret = model.SecretString(strings.TrimSpace(req.Secret))
	}
	if err := model.CreateNotificationChannel(&channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Never echo the secret back.
	channel.Secret = ""
	c.JSON(http.StatusCreated, channel)
}

func findChannel(c *gin.Context) (model.NotificationChannel, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return model.NotificationChannel{}, false
	}
	channel, err := model.GetNotificationChannel(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return model.NotificationChannel{}, false
	}
	return channel, true
}

func UpdateChannel(c *gin.Context) {
	existing, ok := findChannel(c)
	if !ok {
		return
	}
	var req channelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	channel, problem := req.normalize()
	if problem != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": problem})
		return
	}
	channel.ID = existing.ID
	channel.Secret = existing.Secret // keep the stored secret unless replaced below
	if strings.TrimSpace(req.Secret) != "" {
		channel.Secret = model.SecretString(strings.TrimSpace(req.Secret))
	}
	if err := model.UpdateNotificationChannel(&channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	channel.Secret = ""
	c.JSON(http.StatusOK, channel)
}

func DeleteChannel(c *gin.Context) {
	existing, ok := findChannel(c)
	if !ok {
		return
	}
	if err := model.DeleteNotificationChannel(existing.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "channel deleted"})
}

func TestChannel(c *gin.Context) {
	channel, ok := findChannel(c)
	if !ok {
		return
	}
	err := Send(channel, Message{
		Title: "KubeOsprey 通知测试",
		Text:  "这是一条来自 KubeOsprey 的测试通知。如果你看到了这条消息，说明渠道配置正确。",
	})
	if err != nil {
		_ = model.UpdateNotificationChannelDelivery(channel.ID, "failed", err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	_ = model.UpdateNotificationChannelDelivery(channel.ID, "ok", "")
	c.JSON(http.StatusOK, gin.H{"message": "test notification sent"})
}
