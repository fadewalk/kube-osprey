package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/pkg/alertmanager"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
)

const alertManagerRequestTimeout = 15 * time.Second

type AlertManagerHandler struct{}

func NewAlertManagerHandler() *AlertManagerHandler {
	return &AlertManagerHandler{}
}

func (h *AlertManagerHandler) client(c *gin.Context) (*alertmanager.Client, bool) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)
	if !rbac.CanAccessAllNamespaces(user, "alertmanager", string(common.VerbGet), cs.Name) {
		c.JSON(http.StatusForbidden, gin.H{"error": rbac.NoAccess(user.Key(), "get", "alertmanager", common.AllNamespaces, cs.Name)})
		return nil, false
	}
	if cs.AlertClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AlertManager not configured for this cluster"})
		return nil, false
	}
	return cs.AlertClient, true
}

func (h *AlertManagerHandler) writeClient(c *gin.Context, verb common.Verb) (*alertmanager.Client, bool) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)
	if !rbac.CanAccessAllNamespaces(user, "alertmanager", string(verb), cs.Name) {
		c.JSON(http.StatusForbidden, gin.H{"error": rbac.NoAccess(user.Key(), string(verb), "alertmanager", common.AllNamespaces, cs.Name)})
		return nil, false
	}
	if cs.AlertClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AlertManager not configured for this cluster"})
		return nil, false
	}
	return cs.AlertClient, true
}

func (h *AlertManagerHandler) ListAlerts(c *gin.Context) {
	client, ok := h.client(c)
	if !ok {
		return
	}
	ctx, cancel := alertManagerContext(c)
	defer cancel()
	alerts, err := client.ListAlerts(ctx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

func (h *AlertManagerHandler) ListSilences(c *gin.Context) {
	client, ok := h.client(c)
	if !ok {
		return
	}
	ctx, cancel := alertManagerContext(c)
	defer cancel()
	silences, err := client.ListSilences(ctx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, silences)
}

func (h *AlertManagerHandler) CreateSilence(c *gin.Context) {
	client, ok := h.writeClient(c, common.VerbCreate)
	if !ok {
		return
	}
	var payload alertmanager.SilencePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.EndsAt.IsZero() || payload.EndsAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endsAt must be in the future"})
		return
	}
	if payload.StartsAt.IsZero() {
		payload.StartsAt = time.Now()
	}
	if payload.CreatedBy == "" {
		user := c.MustGet("user").(model.User)
		payload.CreatedBy = user.Key()
	}
	ctx, cancel := alertManagerContext(c)
	defer cancel()
	id, err := client.CreateSilence(ctx, payload)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"silenceId": id})
}

func (h *AlertManagerHandler) DeleteSilence(c *gin.Context) {
	client, ok := h.writeClient(c, common.VerbDelete)
	if !ok {
		return
	}
	id := c.Param("silenceId")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "silence id is required"})
		return
	}
	ctx, cancel := alertManagerContext(c)
	defer cancel()
	if err := client.DeleteSilence(ctx, id); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "silence deleted"})
}

func (h *AlertManagerHandler) GetStatus(c *gin.Context) {
	client, ok := h.client(c)
	if !ok {
		return
	}
	ctx, cancel := alertManagerContext(c)
	defer cancel()
	status, err := client.GetStatus(ctx)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func alertManagerContext(c *gin.Context) (ctx context.Context, cancel context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), alertManagerRequestTimeout)
}
