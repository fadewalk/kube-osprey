package notify

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/zxh326/kite/pkg/model"
)

// Event kinds dispatched to subscribed channels. They mirror the constants
// in pkg/model but live here so callers stay short.
const (
	EventPlanPending = model.NotifyEventPlanPending
	EventPlanApplied = model.NotifyEventPlanApplied
	EventPlanFailed  = model.NotifyEventPlanFailed

	EventAlertFiring  = model.NotifyEventAlertFiring
	EventEventWarning = model.NotifyEventEventWarning
)

var dispatchMu sync.Mutex

// HasSubscriber reports whether any enabled channel subscribes to event.
// Polling sources use it to skip work when nobody is listening.
func HasSubscriber(event string) bool {
	channels, err := model.ListEnabledNotificationChannels()
	if err != nil {
		return false
	}
	for _, channel := range channels {
		if channel.SubscribesTo(event) {
			return true
		}
	}
	return false
}

// Dispatch delivers msg to every enabled channel subscribed to event,
// updating each channel's delivery bookkeeping. Failures are logged and
// recorded on the channel — delivery never blocks the caller.
func Dispatch(event, title, text string) {
	channels, err := model.ListEnabledNotificationChannels()
	if err != nil {
		slog.Error("notify: failed to list channels", "event", event, "err", err)
		return
	}
	dispatchMu.Lock()
	defer dispatchMu.Unlock()

	sent := 0
	for _, channel := range channels {
		if !channel.SubscribesTo(event) {
			continue
		}
		err := Send(channel, Message{Title: title, Text: text})
		if err != nil {
			slog.Warn("notify: delivery failed", "channel", channel.Name, "event", event, "err", err)
			status := "failed"
			if uerr := model.UpdateNotificationChannelDelivery(channel.ID, status, err.Error()); uerr != nil {
				slog.Error("notify: failed to record delivery status", "err", uerr)
			}
			continue
		}
		sent++
		if uerr := model.UpdateNotificationChannelDelivery(channel.ID, "ok", ""); uerr != nil {
			slog.Error("notify: failed to record delivery status", "err", uerr)
		}
	}
	if sent > 0 {
		slog.Info("notify: delivered", "event", event, "channels", sent)
	}
}

// PlanEvent is the notification payload for change-plan lifecycle events.
type PlanEvent struct {
	PlanID      uint
	Status      string
	Description string
	Creator     string
	Detail      string // optional: per-step outcome summary
}

// DispatchPlanEvent routes a change-plan lifecycle event to subscribed
// channels with a human-readable title and body.
func DispatchPlanEvent(event string, e PlanEvent) {
	var title string
	switch event {
	case EventPlanPending:
		title = "变更计划待审批"
	case EventPlanApplied:
		title = "变更计划执行成功"
	case EventPlanFailed:
		title = "变更计划执行失败"
	default:
		title = "变更计划更新"
	}
	body := fmt.Sprintf("**%s** #%d\n%s\n创建者: %s", title, e.PlanID, e.Description, e.Creator)
	if e.Status != "" {
		body += fmt.Sprintf("\n状态: %s", e.Status)
	}
	if e.Detail != "" {
		body += fmt.Sprintf("\n%s", e.Detail)
	}
	Dispatch(event, fmt.Sprintf("%s #%d", title, e.PlanID), body)
}
