package changeplan

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
)

type gormDB = gorm.DB

// rollbackOps performs the inverse mutation of one applied step. Injectable
// so the inverse-decision logic is testable without a cluster; op names are
// "delete" | "create" | "restore".
type rollbackOps struct {
	deleteObj  func(ctx context.Context, cs *cluster.ClientSet, obj *unstructured.Unstructured) error
	createObj  func(ctx context.Context, cs *cluster.ClientSet, obj *unstructured.Unstructured) error
	restoreObj func(ctx context.Context, cs *cluster.ClientSet, before *unstructured.Unstructured) error
}

func defaultRollbackOps() rollbackOps {
	return rollbackOps{
		deleteObj: func(ctx context.Context, cs *cluster.ClientSet, obj *unstructured.Unstructured) error {
			return cs.K8sClient.Delete(ctx, obj)
		},
		createObj: func(ctx context.Context, cs *cluster.ClientSet, obj *unstructured.Unstructured) error {
			return cs.K8sClient.Create(ctx, obj)
		},
		restoreObj: func(ctx context.Context, cs *cluster.ClientSet, before *unstructured.Unstructured) error {
			live := before.DeepCopy()
			if err := cs.K8sClient.Get(ctx, client.ObjectKeyFromObject(before), live); err != nil {
				return err
			}
			before.SetResourceVersion(live.GetResourceVersion())
			return cs.K8sClient.Update(ctx, before)
		},
	}
}

// RollbackPlan reverts an APPLIED plan: applied steps are undone in reverse
// order using their before/after snapshots, skipping what never mutated.
// The caller must hold the approve permission (enforced by the handler);
// execution runs against the plan creator's identity semantics unchanged —
// the inverse ops act on the exact objects the steps created or modified.
func (s *Service) RollbackPlan(ctx context.Context, planID uint) (*model.ChangePlan, error) {
	var plan model.ChangePlan
	if err := model.DB.First(&plan, planID).Error; err != nil {
		return nil, err
	}
	if plan.Status != model.ChangePlanApplied {
		return nil, fmt.Errorf("plan %d is not applied (current status: %s)", planID, plan.Status)
	}
	res := model.DB.Model(&model.ChangePlan{}).
		Where("id = ? AND status = ?", planID, model.ChangePlanApplied).
		Update("status", model.ChangePlanRollingBack)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, fmt.Errorf("plan %d is no longer applied", planID)
	}

	if err := model.DB.Preload("Steps", func(db *gormDB) *gormDB { return db.Order("seq DESC") }).First(&plan, planID).Error; err != nil {
		return nil, err
	}

	creator := model.User{Model: model.Model{ID: plan.UserID}, Username: plan.Username}
	failed := false
	for i := range plan.Steps {
		step := &plan.Steps[i]
		if step.ApplyStatus != model.ChangeStepApplied {
			// Never applied: nothing to undo.
			continue
		}
		if failed {
			s.markStep(step, model.ChangeStepFailed, "", "rollback aborted after earlier failure")
			continue
		}
		if err := s.rollbackStep(ctx, &creator, step); err != nil {
			klog.Warningf("Rollback of plan %d step %d failed: %v", planID, step.Seq, err)
			s.markStep(step, model.ChangeStepFailed, "", fmt.Sprintf("rollback failed: %v", err))
			failed = true
			continue
		}
		s.markStep(step, model.ChangeStepRolledBack, "", "")
	}

	finalStatus := model.ChangePlanRolledBack
	if failed {
		finalStatus = model.ChangePlanRollbackFailed
	}
	if err := model.DB.Model(&model.ChangePlan{}).Where("id = ?", planID).
		Updates(map[string]any{"status": finalStatus}).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Preload("Steps", func(db *gormDB) *gormDB { return db.Order("seq ASC") }).First(&plan, planID).Error; err != nil {
		return nil, err
	}
	if failed {
		return &plan, fmt.Errorf("one or more steps failed to roll back")
	}
	return &plan, nil
}

// rollbackStep undoes one applied step using its snapshots:
//
//	create (before absent)      -> delete the created object
//	delete (after absent)       -> recreate from the before snapshot
//	update/apply/patch          -> restore the before snapshot over the live object
func (s *Service) rollbackStep(ctx context.Context, creator *model.User, step *model.ChangePlanStep) error {
	var args map[string]any
	_ = json.Unmarshal([]byte(step.ArgsJSON), &args)

	cs, err := s.clientSet(step.Cluster)
	if err != nil {
		return err
	}

	switch step.ToolName {
	case "create_resource":
		// The object did not exist before this step: remove it.
		obj, err := snapshotObject(step.AfterJSON)
		if err != nil {
			return err
		}
		if obj == nil {
			return fmt.Errorf("step has no after snapshot to delete")
		}
		return s.rollback.deleteObj(ctx, cs, obj)

	case "delete_resource":
		// The object existed before this step: bring it back.
		obj, err := snapshotObject(step.BeforeJSON)
		if err != nil {
			return err
		}
		if obj == nil {
			return fmt.Errorf("step has no before snapshot to restore")
		}
		obj.SetResourceVersion("")
		return s.rollback.createObj(ctx, cs, obj)

	default: // update / apply / patch
		before, err := snapshotObject(step.BeforeJSON)
		if err != nil {
			return err
		}
		if before == nil {
			return fmt.Errorf("step has no before snapshot to restore")
		}
		return s.rollback.restoreObj(ctx, cs, before)
	}
}

func snapshotObject(raw string) (*unstructured.Unstructured, error) {
	if raw == "" {
		return nil, nil
	}
	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal([]byte(raw), obj); err != nil {
		return nil, err
	}
	if obj.GetName() == "" {
		return nil, fmt.Errorf("snapshot has no name")
	}
	return obj, nil
}
