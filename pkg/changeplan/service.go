// Package changeplan implements the plan-based approval workflow for AI
// mutations: agents may only author plans (with server-side dry-runs), humans
// approve or reject in the UI, and execution happens exclusively through
// apply with a one-time approval token.
package changeplan

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/zxh326/kite/pkg/ai"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
)

// DefaultServiceTTL matches the model default; overridable in tests.
const DefaultServiceTTL = model.DefaultChangePlanTTL

type Service struct {
	cm *cluster.ClusterManager
	// dryRun is injectable so state-machine tests run without a cluster.
	dryRun func(ctx context.Context, cs *cluster.ClientSet, toolName string, args map[string]any) (before, after map[string]any, supported bool, err error)
	// execute defaults to ai.ExecuteMutation; injectable in tests.
	execute func(ctx context.Context, cs *cluster.ClientSet, user model.User, toolName string, args map[string]any) (string, error)
	// authorize defaults to ai.AuthorizeToolForUser; injectable in tests.
	authorize func(ctx context.Context, cs *cluster.ClientSet, user model.User, toolName string, args map[string]any) error
	// clientSet defaults to cm.GetClientSet; injectable in tests.
	clientSet func(name string) (*cluster.ClientSet, error)
	// rollback holds the inverse mutations; injectable in tests.
	rollback rollbackOps
	ttl      time.Duration
}

func NewService(cm *cluster.ClusterManager) *Service {
	return &Service{
		cm:        cm,
		dryRun:    ai.DryRunMutation,
		execute:   ai.ExecuteMutation,
		authorize: ai.AuthorizeToolForUser,
		clientSet: cm.GetClientSet,
		rollback:  defaultRollbackOps(),
		ttl:       DefaultServiceTTL,
	}
}

func generateApprovalToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func dangerForTool(toolName string) string {
	if toolName == "delete_resource" {
		return model.DangerLevelHigh
	}
	return model.DangerLevelMedium
}

// CreatePlan opens an empty plan in CREATED state. It is invisible to the
// approval center until Finalize; the approval token is returned exactly once.
func (s *Service) CreatePlan(user model.User, clientName, agentSessionID, description string) (*model.ChangePlan, string, error) {
	token, err := generateApprovalToken()
	if err != nil {
		return nil, "", fmt.Errorf("generating approval token: %w", err)
	}
	plan := &model.ChangePlan{
		UserID:         user.ID,
		Username:       user.Username,
		ClientName:     clientName,
		AgentSessionID: agentSessionID,
		Description:    description,
		Status:         model.ChangePlanCreated,
		DangerLevel:    model.DangerLevelLow,
		ApprovalToken:  model.SecretString(token),
	}
	if err := model.DB.Create(plan).Error; err != nil {
		return nil, "", err
	}
	return plan, token, nil
}

// AppendStep validates and dry-runs one mutation, attaching it to a CREATED
// plan. RBAC is checked against the plan creator here and again at apply time.
func (s *Service) AppendStep(ctx context.Context, user model.User, planID uint, toolName, description string, args map[string]any) (*model.ChangePlanStep, error) {
	if !ai.IsPlanMutationTool(toolName) {
		return nil, fmt.Errorf("tool %q is not allowed in change plans", toolName)
	}
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.ChangePlan{}).
			Where("id = ? AND user_id = ? AND status = ?", planID, user.ID, model.ChangePlanCreated).
			Update("updated_at", time.Now())
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("plan %d is not open for steps (wrong state or owner)", planID)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	cs, err := s.clientSet(mapString(args, "cluster"))
	if err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, cs, user, toolName, args); err != nil {
		return nil, err
	}

	step := &model.ChangePlanStep{
		PlanID:      planID,
		ToolName:    toolName,
		Description: description,
		Cluster:     cs.Name,
		DangerLevel: dangerForTool(toolName),
		Status:      model.ChangeStepPending,
	}
	if raw, err := json.Marshal(args); err == nil {
		step.ArgsJSON = string(raw)
	}

	before, after, supported, dryErr := s.dryRun(ctx, cs, toolName, args)
	step.DryRunSupported = supported
	switch {
	case !supported:
		step.DryRunStatus = model.ChangeStepDryRunUnsupported
	case dryErr != nil:
		step.DryRunStatus = model.ChangeStepDryRunFailed
		step.ErrorMessage = dryErr.Error()
	default:
		step.DryRunStatus = model.ChangeStepDryRunOK
	}
	step.BeforeJSON = marshalSnapshot(before)
	step.AfterJSON = marshalSnapshot(after)
	if dryErr != nil && supported {
		step.DryRunResponse = dryErr.Error()
	}

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.ChangePlanStep{}).Where("plan_id = ?", planID).Count(&count).Error; err != nil {
			return err
		}
		step.Seq = int(count) + 1
		if err := tx.Create(step).Error; err != nil {
			return err
		}
		return tx.Model(&model.ChangePlan{}).Where("id = ?", planID).
			Update("danger_level", model.MaxDangerLevel(dangerForTool(toolName), s.planDanger(tx, planID))).Error
	}); err != nil {
		return nil, err
	}
	return step, nil
}

func (s *Service) planDanger(tx *gorm.DB, planID uint) string {
	var plan model.ChangePlan
	if err := tx.Select("danger_level").First(&plan, planID).Error; err != nil {
		return model.DangerLevelLow
	}
	return plan.DangerLevel
}

// FinalizePlan atomically moves CREATED -> PENDING, making the plan visible
// to approvers. The CAS on status makes concurrent appends fail once finalized.
func (s *Service) FinalizePlan(user model.User, planID uint) error {
	var count int64
	if err := model.DB.Model(&model.ChangePlanStep{}).Where("plan_id = ?", planID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("plan %d has no steps", planID)
	}
	expires := time.Now().Add(s.ttl)
	res := model.DB.Model(&model.ChangePlan{}).
		Where("id = ? AND user_id = ? AND status = ?", planID, user.ID, model.ChangePlanCreated).
		Updates(map[string]any{"status": model.ChangePlanPending, "expires_at": expires})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("plan %d is not in CREATED state or not owned by you", planID)
	}
	klog.V(1).Infof("Change plan %d finalized by %s", planID, user.Username)
	return nil
}

func (s *Service) expireStalePlans() {
	model.DB.Model(&model.ChangePlan{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", model.ChangePlanPending, time.Now()).
		Update("status", model.ChangePlanExpired)
}

// ApprovePlan transitions PENDING -> APPROVED. The approver must differ from
// the creator; expired plans flip to EXPIRED instead.
func (s *Service) ApprovePlan(approver model.User, planID uint) error {
	s.expireStalePlans()

	var plan model.ChangePlan
	if err := model.DB.First(&plan, planID).Error; err != nil {
		return err
	}
	if plan.UserID == approver.ID {
		return fmt.Errorf("the plan creator cannot approve their own plan")
	}
	res := model.DB.Model(&model.ChangePlan{}).
		Where("id = ? AND status = ?", planID, model.ChangePlanPending).
		Updates(map[string]any{
			"status":           model.ChangePlanApproved,
			"approver_user_id": approver.ID,
			"approver_name":    approver.Username,
			"approved_at":      time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("plan %d is not pending approval", planID)
	}
	klog.Infof("Change plan %d approved by %s", planID, approver.Username)
	// Reload so the chat auto-apply sees the post-CAS APPROVED state.
	var fresh model.ChangePlan
	if err := model.DB.First(&fresh, planID).Error; err == nil {
		s.autoApplyChatPlan(fresh)
	}
	return nil
}

// RejectPlan transitions PENDING -> REJECTED with the same separation rule.
func (s *Service) RejectPlan(approver model.User, planID uint) error {
	s.expireStalePlans()

	var plan model.ChangePlan
	if err := model.DB.First(&plan, planID).Error; err != nil {
		return err
	}
	if plan.UserID == approver.ID {
		return fmt.Errorf("the plan creator cannot reject their own plan")
	}
	res := model.DB.Model(&model.ChangePlan{}).
		Where("id = ? AND status = ?", planID, model.ChangePlanPending).
		Updates(map[string]any{
			"status":           model.ChangePlanRejected,
			"approver_user_id": approver.ID,
			"approver_name":    approver.Username,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("plan %d is not pending approval", planID)
	}
	klog.Infof("Change plan %d rejected by %s", planID, approver.Username)
	return nil
}

// ApplyPlan executes an APPROVED plan step by step as the plan creator. The
// token is consumed atomically with the APPROVED -> APPLYING transition, so
// replays and double-applies fail. RBAC is re-checked per step inside
// ai.ExecuteMutation in case permissions changed since approval.
func (s *Service) ApplyPlan(ctx context.Context, user model.User, planID uint, token string) (*model.ChangePlan, error) {
	s.expireStalePlans()

	// The token is stored encrypted (SecretString), so it is compared in Go
	// and consumed with a state CAS that only one caller can win.
	var plan model.ChangePlan
	if err := model.DB.First(&plan, planID).Error; err != nil {
		return nil, err
	}
	if plan.Status != model.ChangePlanApproved {
		return nil, fmt.Errorf("plan %d is not approved (current status: %s)", planID, plan.Status)
	}
	if subtle.ConstantTimeCompare([]byte(string(plan.ApprovalToken)), []byte(token)) != 1 || token == "" {
		return nil, fmt.Errorf("invalid approval token for plan %d", planID)
	}
	res := model.DB.Model(&model.ChangePlan{}).
		Where("id = ? AND status = ? AND approval_token <> ''", planID, model.ChangePlanApproved).
		Updates(map[string]any{"status": model.ChangePlanApplying, "approval_token": ""})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, fmt.Errorf("plan %d was already applied or its token was consumed", planID)
	}
	return s.applyPlanSteps(ctx, planID, user)
}

// applyPlanSteps runs an APPLYING/APPROVED plan's steps in order as the plan
// creator, stopping at the first failure. Used by the token flow and by the
// chat-plan auto-apply after approval.
func (s *Service) applyPlanSteps(ctx context.Context, planID uint, creator model.User) (*model.ChangePlan, error) {
	var plan model.ChangePlan
	if err := model.DB.First(&plan, planID).Error; err != nil {
		return nil, err
	}
	if plan.Status == model.ChangePlanApproved {
		res := model.DB.Model(&model.ChangePlan{}).
			Where("id = ? AND status = ?", planID, model.ChangePlanApproved).
			Update("status", model.ChangePlanApplying)
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, fmt.Errorf("plan %d was already applied or its token was consumed", planID)
		}
	}

	if err := model.DB.Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).First(&plan, planID).Error; err != nil {
		return nil, err
	}

	failed := false
	for i := range plan.Steps {
		step := &plan.Steps[i]
		if failed {
			s.markStep(step, model.ChangeStepSkipped, "", "skipped after earlier failure")
			continue
		}
		cs, err := s.clientSet(step.Cluster)
		if err != nil {
			s.markStep(step, model.ChangeStepFailed, "", err.Error())
			failed = true
			continue
		}
		var args map[string]any
		if err := json.Unmarshal([]byte(step.ArgsJSON), &args); err != nil {
			s.markStep(step, model.ChangeStepFailed, "", fmt.Sprintf("corrupt step args: %v", err))
			failed = true
			continue
		}
		result, err := s.execute(ctx, cs, creator, step.ToolName, args)
		if err != nil {
			s.markStep(step, model.ChangeStepFailed, result, err.Error())
			failed = true
			continue
		}
		s.markStep(step, model.ChangeStepApplied, result, "")
	}

	finalStatus := model.ChangePlanApplied
	errMsg := ""
	if failed {
		finalStatus = model.ChangePlanFailed
		errMsg = "one or more steps failed"
	}
	if err := model.DB.Model(&model.ChangePlan{}).Where("id = ?", planID).
		Updates(map[string]any{"status": finalStatus, "approval_token": ""}).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).First(&plan, planID).Error; err != nil {
		return nil, err
	}
	if failed {
		return &plan, fmt.Errorf("%s", errMsg)
	}
	return &plan, nil
}

func (s *Service) markStep(step *model.ChangePlanStep, status, response, errMsg string) {
	updates := map[string]any{
		"apply_status": status,
		"status":       status,
	}
	if response != "" {
		updates["apply_response"] = response
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	if status == model.ChangeStepApplied {
		now := time.Now()
		updates["applied_at"] = &now
	}
	if err := model.DB.Model(step).Updates(updates).Error; err != nil {
		klog.Errorf("Failed to update change plan step %d: %v", step.ID, err)
	}
	step.ApplyStatus = status
	step.Status = status
}

// GetPlanForUser returns the plan when the user created it or holds the
// approve permission; otherwise it is hidden.
func (s *Service) GetPlanForUser(planID uint, user model.User, canApprove bool) (*model.ChangePlan, error) {
	s.expireStalePlans()
	var plan model.ChangePlan
	if err := model.DB.Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).First(&plan, planID).Error; err != nil {
		return nil, err
	}
	if plan.UserID != user.ID && !canApprove {
		return nil, gorm.ErrRecordNotFound
	}
	return &plan, nil
}

// ListPlansForUser returns plans; approvers see everything, creators only
// their own.
func (s *Service) ListPlansForUser(user model.User, canApprove bool, status string) ([]model.ChangePlan, error) {
	s.expireStalePlans()
	query := model.DB
	if !canApprove {
		query = query.Where("user_id = ?", user.ID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var plans []model.ChangePlan
	err := query.Order("updated_at DESC").Limit(100).
		Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Find(&plans).Error
	return plans, err
}

func mapString(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func marshalSnapshot(v map[string]any) string {
	if v == nil {
		return ""
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(raw)
}
