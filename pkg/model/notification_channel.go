package model

import (
	"time"
)

// Notification event kinds (stored csv on the channel).
const (
	NotifyEventPlanPending = "plan_pending" // a change plan awaits approval
	NotifyEventPlanApplied = "plan_applied" // a plan finished successfully
	NotifyEventPlanFailed  = "plan_failed"  // a plan finished with failures
)

// NotificationChannel is an outbound webhook-style sink (DingTalk, WeCom,
// Feishu, Slack or a generic receiver) that receives notification messages
// for the events it subscribes to.
type NotificationChannel struct {
	Model
	Name     string       `json:"name" gorm:"column:name;type:varchar(100);not null;uniqueIndex"`
	Type     string       `json:"type" gorm:"column:type;type:varchar(20);not null"` // dingtalk | wecom | feishu | slack | webhook
	Endpoint string       `json:"endpoint" gorm:"column:endpoint;type:varchar(2048);not null"`
	Secret   SecretString `json:"-" gorm:"column:secret;type:text"`
	Events   string       `json:"events" gorm:"column:events;type:varchar(200);not null;default:'plan_pending'"`
	Enabled  bool         `json:"enabled" gorm:"column:enabled;not null;default:true"`
	// Delivery bookkeeping, updated by the dispatcher.
	LastStatus string     `json:"lastStatus" gorm:"column:last_status;type:varchar(20);default:''"`
	LastError  string     `json:"lastError" gorm:"column:last_error;type:varchar(500);default:''"`
	LastSentAt *time.Time `json:"lastSentAt"`
}

func (NotificationChannel) TableName() string { return "notification_channels" }

// SubscribesTo reports whether the channel should receive an event kind.
func (c NotificationChannel) SubscribesTo(event string) bool {
	return c.Enabled && containsCSV(c.Events, event)
}

func containsCSV(csv, needle string) bool {
	start := 0
	for i := 0; i <= len(csv); i++ {
		if i == len(csv) || csv[i] == ',' {
			if trim(csv[start:i]) == needle {
				return true
			}
			start = i + 1
		}
	}
	return false
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func ListNotificationChannels() ([]NotificationChannel, error) {
	var channels []NotificationChannel
	if err := DB.Order("name ASC").Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func ListEnabledNotificationChannels() ([]NotificationChannel, error) {
	var channels []NotificationChannel
	if err := DB.Where("enabled = ?", true).Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func GetNotificationChannel(id uint) (NotificationChannel, error) {
	var channel NotificationChannel
	err := DB.First(&channel, id).Error
	return channel, err
}

func CreateNotificationChannel(channel *NotificationChannel) error {
	return DB.Create(channel).Error
}

func UpdateNotificationChannel(channel *NotificationChannel) error {
	return DB.Save(channel).Error
}

func UpdateNotificationChannelDelivery(id uint, status, lastErr string) error {
	now := time.Now()
	return DB.Model(&NotificationChannel{}).Where("id = ?", id).
		Updates(map[string]any{"last_status": status, "last_error": lastErr, "last_sent_at": now}).Error
}

func DeleteNotificationChannel(id uint) error {
	return DB.Delete(&NotificationChannel{}, id).Error
}
