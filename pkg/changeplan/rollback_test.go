package changeplan

import (
	"context"
	"strings"
	"testing"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TestRollbackInverseOps drives the full rollback of a plan whose steps were
// create + delete, asserting reverse order and the right inverse per tool.
func TestRollbackInverseOps(t *testing.T) {
	setupPlanDB(t)
	svc := newTestService()

	// Drive a plan to APPLIED with create (applied) + delete (applied).
	svc.dryRun = func(_ context.Context, _ *cluster.ClientSet, tool string, _ map[string]any) (map[string]any, map[string]any, bool, error) {
		if tool == "delete_resource" {
			return map[string]any{"kind": "ConfigMap", "apiVersion": "v1", "metadata": map[string]any{"name": "old-cm"}}, nil, false, nil // unsupported, snapshot kept
		}
		return map[string]any{"old": true}, map[string]any{"kind": "ConfigMap", "apiVersion": "v1", "metadata": map[string]any{"name": "rb-cm"}}, true, nil
	}
	user := creatorUser(1)
	plan, token, err := svc.CreatePlan(user, "test", "", "rollback smoke")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	var ops []string
	svc.execute = func(_ context.Context, _ *cluster.ClientSet, _ model.User, tool string, _ map[string]any) (string, error) {
		ops = append(ops, "apply:"+tool)
		return "ok", nil
	}
	cmYAML := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: rb-cm\n  namespace: default\ndata:\n  k: v\n"
	if _, err := svc.AppendStep(context.Background(), user, plan.ID, "create_resource", "", map[string]any{"yaml": cmYAML}); err != nil {
		t.Fatalf("AppendStep create: %v", err)
	}
	if _, err := svc.AppendStep(context.Background(), user, plan.ID, "delete_resource", "", map[string]any{"kind": "ConfigMap", "name": "old-cm", "namespace": "default"}); err != nil {
		t.Fatalf("AppendStep delete: %v", err)
	}
	if err := svc.FinalizePlan(user, plan.ID); err != nil {
		t.Fatalf("FinalizePlan: %v", err)
	}
	if err := svc.ApprovePlan(approverUser(), plan.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	if _, err := svc.ApplyPlan(context.Background(), user, plan.ID, token); err != nil {
		t.Fatalf("ApplyPlan: %v", err)
	}

	// Rollback: expect reverse order — undo delete (recreate old-cm), then
	// undo create (delete rb-cm).
	var rollbackCalls []string
	svc.rollback = rollbackOps{
		deleteObj: func(_ context.Context, _ *cluster.ClientSet, obj *unstructured.Unstructured) error {
			rollbackCalls = append(rollbackCalls, "delete:"+obj.GetName())
			return nil
		},
		createObj: func(_ context.Context, _ *cluster.ClientSet, obj *unstructured.Unstructured) error {
			rollbackCalls = append(rollbackCalls, "create:"+obj.GetName())
			return nil
		},
		restoreObj: func(_ context.Context, _ *cluster.ClientSet, before *unstructured.Unstructured) error {
			rollbackCalls = append(rollbackCalls, "restore:"+before.GetName())
			return nil
		},
	}

	rolled, rbErr := svc.RollbackPlan(context.Background(), plan.ID)
	if rbErr != nil {
		t.Fatalf("RollbackPlan: %v", rbErr)
	}
	if rolled.Status != model.ChangePlanRolledBack {
		t.Fatalf("status = %s, want ROLLED_BACK", rolled.Status)
	}
	want := []string{"create:old-cm", "delete:rb-cm"}
	if len(rollbackCalls) != len(want) {
		t.Fatalf("rollback calls = %v, want %v", rollbackCalls, want)
	}
	for i := range want {
		if rollbackCalls[i] != want[i] {
			t.Fatalf("rollback call %d = %s, want %s", i, rollbackCalls[i], want[i])
		}
	}
	for _, step := range rolled.Steps {
		if step.Status != model.ChangeStepRolledBack {
			t.Fatalf("step %d status = %s, want ROLLED_BACK", step.Seq, step.Status)
		}
	}

	// Double rollback must fail.
	if _, err := svc.RollbackPlan(context.Background(), plan.ID); err == nil {
		t.Fatal("second rollback must fail")
	}
}

// TestRollbackFailurePartial covers a mid-rollback failure: remaining steps
// are marked FAILED, plan lands ROLLBACK_FAILED.
func TestRollbackFailurePartial(t *testing.T) {
	setupPlanDB(t)
	svc := newTestService()

	user := creatorUser(1)
	plan, token, _ := svc.CreatePlan(user, "", "", "d")
	for i := 0; i < 2; i++ {
		if _, err := svc.AppendStep(context.Background(), user, plan.ID, "create_resource", "", map[string]any{"yaml": "x"}); err != nil {
			t.Fatalf("AppendStep: %v", err)
		}
	}
	if err := svc.FinalizePlan(user, plan.ID); err != nil {
		t.Fatalf("FinalizePlan: %v", err)
	}
	if err := svc.ApprovePlan(approverUser(), plan.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	svc.execute = func(context.Context, *cluster.ClientSet, model.User, string, map[string]any) (string, error) {
		return "ok", nil
	}
	if _, err := svc.ApplyPlan(context.Background(), user, plan.ID, token); err != nil {
		t.Fatalf("ApplyPlan: %v", err)
	}

	svc.rollback = rollbackOps{
		deleteObj: func(_ context.Context, _ *cluster.ClientSet, obj *unstructured.Unstructured) error {
			return context.DeadlineExceeded
		},
	}
	rolled, rbErr := svc.RollbackPlan(context.Background(), plan.ID)
	if rbErr == nil {
		t.Fatal("expected rollback error")
	}
	if rolled.Status != model.ChangePlanRollbackFailed {
		t.Fatalf("status = %s, want ROLLBACK_FAILED", rolled.Status)
	}
	statuses := map[int]string{}
	for _, s := range rolled.Steps {
		statuses[s.Seq] = s.Status
		if !strings.Contains(s.ErrorMessage, "rollback failed") && s.Seq > 1 {
			t.Fatalf("step %d error = %q", s.Seq, s.ErrorMessage)
		}
	}
	if statuses[1] != model.ChangeStepFailed || statuses[2] != model.ChangeStepFailed {
		t.Fatalf("step statuses = %v", statuses)
	}

	// Guard: only APPLIED plans can roll back.
	if _, err := svc.RollbackPlan(context.Background(), plan.ID); err == nil {
		t.Fatal("rollback of non-applied plan must fail")
	}
}

// TestRollbackGuards verifies permission handling happens at the handler and
// never-applied steps are untouched by rollback.
func TestRollbackGuards(t *testing.T) {
	setupPlanDB(t)
	svc := newTestService()
	user := creatorUser(1)
	plan, token, _ := svc.CreatePlan(user, "", "", "d")
	for i := 0; i < 2; i++ {
		if _, err := svc.AppendStep(context.Background(), user, plan.ID, "create_resource", "", map[string]any{"yaml": "x"}); err != nil {
			t.Fatalf("AppendStep: %v", err)
		}
	}
	if err := svc.FinalizePlan(user, plan.ID); err != nil {
		t.Fatalf("FinalizePlan: %v", err)
	}
	if err := svc.ApprovePlan(approverUser(), plan.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	if _, err := svc.ApplyPlan(context.Background(), user, plan.ID, token); err != nil {
		t.Fatalf("ApplyPlan: %v", err)
	}

	// Simulate a step that never mutated (e.g. from a partial apply): the
	// second step is marked SKIPPED and must not be rolled back.
	model.DB.Model(&model.ChangePlanStep{}).Where("plan_id = ? AND seq = ?", plan.ID, 2).
		Updates(map[string]any{"apply_status": model.ChangeStepSkipped, "status": model.ChangeStepSkipped})

	calls := 0
	svc.rollback = rollbackOps{
		deleteObj: func(context.Context, *cluster.ClientSet, *unstructured.Unstructured) error {
			calls++
			return nil
		},
	}
	if _, err := svc.RollbackPlan(context.Background(), plan.ID); err != nil {
		t.Fatalf("RollbackPlan: %v", err)
	}
	if calls != 1 {
		t.Fatalf("rollback touched %d objects, want 1 (skipped step untouched)", calls)
	}
}
