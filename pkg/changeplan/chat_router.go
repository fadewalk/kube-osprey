package changeplan

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/zxh326/kite/pkg/ai"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
)

// ChatClientName marks plans created by the embedded AI assistant. These
// plans auto-apply on approval: the assistant cannot poll status and apply
// with a token like MCP agents do, so the human approval itself is the
// trigger and execution re-checks the creator's RBAC per step.
const ChatClientName = "kite-chat"

// WebhookClientName marks plans created by CI/CD webhooks. Like chat plans
// they auto-apply on approval: the human approval is the gate, and the
// upgrade step re-checks the creator's RBAC at execution time.
const WebhookClientName = "webhook"

type chatRouter struct {
	svc *Service
}

// ChatRouter adapts the service to the ai.ChangePlanRouter interface.
func (s *Service) ChatRouter() ai.ChangePlanRouter { return chatRouter{s} }

// SubmitMutation implements ai.ChangePlanRouter.
func (r chatRouter) SubmitMutation(ctx context.Context, cs *cluster.ClientSet, user model.User, toolName string, args map[string]interface{}, sessionID string) (string, error) {
	description := fmt.Sprintf("AI assistant: %s", toolName)
	plan, _, err := r.svc.CreatePlan(user, ChatClientName, sessionID, description)
	if err != nil {
		return "", err
	}
	if _, err := r.svc.AppendStep(ctx, user, plan.ID, toolName, description, args); err != nil {
		// Dry-run or validation failure: bury the plan so it does not linger
		// as an invisible CREATED row, and surface the reason to the model.
		model.DB.Model(&model.ChangePlan{}).Where("id = ?", plan.ID).
			Update("status", model.ChangePlanFailed)
		return "", err
	}
	if err := r.svc.FinalizePlan(user, plan.ID); err != nil {
		return "", err
	}
	klog.Infof("Chat assistant submitted change plan %d (%s) by %s", plan.ID, toolName, user.Username)
	return fmt.Sprintf(
		"Submitted as change plan #%d and pending human approval in the Change Plans page (dry-run passed). It applies automatically once approved. Do not retry the mutation; inform the user to approve the plan.",
		plan.ID,
	), nil
}

// autoApplyChatPlan applies an approved chat plan in the background; the
// approval itself was the human gate.
func (s *Service) autoApplyChatPlan(plan model.ChangePlan) {
	if plan.Status != model.ChangePlanApproved {
		return
	}
	if plan.ClientName != ChatClientName && plan.ClientName != WebhookClientName {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		creator := model.User{Model: model.Model{ID: plan.UserID}, Username: plan.Username}
		if _, err := s.applyPlanSteps(ctx, plan.ID, creator); err != nil {
			klog.Warningf("Plan %d auto-apply failed: %v", plan.ID, err)
		} else {
			klog.Infof("Plan %d auto-applied after approval", plan.ID)
		}
	}()
}
