package changeplan

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupPlanDB(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.ChangePlan{}, &model.ChangePlanStep{}); err != nil {
		t.Fatalf("migrating test database: %v", err)
	}
	// Serialize access so concurrent-finalize tests are deterministic.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	model.DB = db
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = originalDB
	})
}

func newTestService() *Service {
	svc := NewService(&cluster.ClusterManager{})
	svc.dryRun = func(context.Context, *cluster.ClientSet, string, map[string]any) (map[string]any, map[string]any, bool, error) {
		snap := map[string]any{"kind": "ConfigMap", "apiVersion": "v1", "metadata": map[string]any{"name": "cm"}}
		return snap, map[string]any{"kind": "ConfigMap", "apiVersion": "v1", "metadata": map[string]any{"name": "cm-new"}}, true, nil
	}
	// RBAC-at-append semantics are covered by the ai package tests; here the
	// state machine is what's under test.
	svc.authorize = func(context.Context, *cluster.ClientSet, model.User, string, map[string]any) error {
		return nil
	}
	svc.clientSet = func(string) (*cluster.ClientSet, error) {
		return &cluster.ClientSet{Name: "test"}, nil
	}
	svc.execute = func(context.Context, *cluster.ClientSet, model.User, string, map[string]any) (string, error) {
		return "ok", nil
	}
	return svc
}

func creatorUser(id uint) model.User {
	return model.User{Model: model.Model{ID: id}, Username: "agent"}
}

func approverUser() model.User {
	return model.User{
		Model:    model.Model{ID: 999},
		Username: "human",
		Roles: []common.Role{{
			Name: "approver", Clusters: []string{"*"}, Namespaces: []string{"*"},
			Resources: []string{"*"}, Verbs: []string{"*"},
		}},
	}
}

// buildOpenPlan drives a plan to the given lifecycle stage.
func buildOpenPlan(t *testing.T, svc *Service, stage string) (*model.ChangePlan, string) {
	t.Helper()
	user := creatorUser(1)
	plan, token, err := svc.CreatePlan(user, "test-client", "", "smoke plan")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	if stage == "CREATED" {
		return plan, token
	}
	if _, err := svc.AppendStep(context.Background(), user, plan.ID, "create_resource", "add configmap", map[string]any{
		"yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm\n  namespace: default\n",
	}); err != nil {
		t.Fatalf("AppendStep: %v", err)
	}
	if err := svc.FinalizePlan(user, plan.ID); err != nil {
		t.Fatalf("FinalizePlan: %v", err)
	}
	if stage == "PENDING" {
		return plan, token
	}
	if err := svc.ApprovePlan(approverUser(), plan.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	return plan, token
}

// TestPlanStateMachineHappyPath drives CREATED -> PENDING -> APPROVED ->
// APPLYING -> APPLIED and asserts every step lands APPLIED.
func TestPlanStateMachineHappyPath(t *testing.T) {
	setupPlanDB(t)
	svc := newTestService()
	var executed []string
	svc.execute = func(_ context.Context, _ *cluster.ClientSet, _ model.User, tool string, _ map[string]any) (string, error) {
		executed = append(executed, tool)
		return "ok", nil
	}

	user := creatorUser(1)
	plan, token, err := svc.CreatePlan(user, "c", "", "d")
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	if plan.Status != model.ChangePlanCreated {
		t.Fatalf("initial status = %s", plan.Status)
	}
	if len(token) != 64 {
		t.Fatalf("token length = %d, want 64 hex chars", len(token))
	}

	for i := 0; i < 2; i++ {
		if _, err := svc.AppendStep(context.Background(), user, plan.ID, "create_resource", "", map[string]any{"yaml": "x"}); err != nil {
			t.Fatalf("AppendStep %d: %v", i, err)
		}
	}
	if err := svc.FinalizePlan(user, plan.ID); err != nil {
		t.Fatalf("FinalizePlan: %v", err)
	}
	var stored model.ChangePlan
	model.DB.First(&stored, plan.ID)
	if stored.Status != model.ChangePlanPending || stored.ExpiresAt == nil {
		t.Fatalf("after finalize: status=%s expires=%v", stored.Status, stored.ExpiresAt)
	}

	if err := svc.ApprovePlan(approverUser(), plan.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	model.DB.First(&stored, plan.ID)
	if stored.Status != model.ChangePlanApproved || stored.ApproverName != "human" {
		t.Fatalf("after approve: status=%s approver=%s", stored.Status, stored.ApproverName)
	}

	applied, err := svc.ApplyPlan(context.Background(), user, plan.ID, token)
	if err != nil {
		t.Fatalf("ApplyPlan: %v", err)
	}
	if applied.Status != model.ChangePlanApplied {
		t.Fatalf("final status = %s", applied.Status)
	}
	for _, step := range applied.Steps {
		if step.Status != model.ChangeStepApplied {
			t.Fatalf("step %d status = %s", step.Seq, step.Status)
		}
	}
	if len(executed) != 2 {
		t.Fatalf("executed %d steps, want 2", len(executed))
	}

	// The token must be consumed after apply.
	model.DB.First(&stored, plan.ID)
	if stored.ApprovalToken != "" {
		t.Fatal("approval token was not consumed")
	}
	if _, err := svc.ApplyPlan(context.Background(), user, plan.ID, token); err == nil {
		t.Fatal("token replay must be rejected")
	}
}

// TestPlanStateMachineRejectAndFailPaths covers REJECTED, APPLYING -> FAILED
// with remaining steps SKIPPED, and creation guards.
func TestPlanStateMachineRejectAndFailPaths(t *testing.T) {
	setupPlanDB(t)
	svc := newTestService()

	creator := creatorUser(1)
	// Rejected path.
	plan, _ := buildOpenPlan(t, svc, "PENDING")
	if err := svc.RejectPlan(approverUser(), plan.ID); err != nil {
		t.Fatalf("RejectPlan: %v", err)
	}
	var stored model.ChangePlan
	model.DB.First(&stored, plan.ID)
	if stored.Status != model.ChangePlanRejected {
		t.Fatalf("status = %s, want REJECTED", stored.Status)
	}
	if err := svc.ApprovePlan(approverUser(), plan.ID); err == nil {
		t.Fatal("approve after reject must fail")
	}

	// Failed apply: second step fails, third must be SKIPPED.
	svc2 := newTestService()
	calls := 0
	svc2.execute = func(context.Context, *cluster.ClientSet, model.User, string, map[string]any) (string, error) {
		calls++
		if calls == 2 {
			return "", errors.New("boom")
		}
		return "ok", nil
	}
	plan2, token2, _ := svc2.CreatePlan(creator, "", "", "d")
	for i := 0; i < 3; i++ {
		if _, err := svc2.AppendStep(context.Background(), creator, plan2.ID, "create_resource", "", map[string]any{"yaml": "x"}); err != nil {
			t.Fatalf("AppendStep: %v", err)
		}
	}
	if err := svc2.FinalizePlan(creator, plan2.ID); err != nil {
		t.Fatalf("FinalizePlan: %v", err)
	}
	if err := svc2.ApprovePlan(approverUser(), plan2.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	applied, applyErr := svc2.ApplyPlan(context.Background(), creator, plan2.ID, token2)
	if applyErr == nil {
		t.Fatal("expected apply error")
	}
	if applied.Status != model.ChangePlanFailed {
		t.Fatalf("plan status = %s, want FAILED", applied.Status)
	}
	statuses := map[int]string{}
	for _, s := range applied.Steps {
		statuses[s.Seq] = s.Status
		t.Logf("step %d tool=%s status=%s err=%q applyResp=%q", s.Seq, s.ToolName, s.Status, s.ErrorMessage, s.ApplyResponse)
	}
	if statuses[1] != model.ChangeStepApplied || statuses[2] != model.ChangeStepFailed || statuses[3] != model.ChangeStepSkipped {
		t.Fatalf("step statuses = %v", statuses)
	}
}

// TestPlanStateMachineGuards covers creator==approver, expiry, finalize
// guards, and concurrent finalize.
func TestPlanStateMachineGuards(t *testing.T) {
	setupPlanDB(t)
	svc := newTestService()
	creator := creatorUser(1)

	t.Run("creator cannot approve or reject", func(t *testing.T) {
		plan, _ := buildOpenPlan(t, svc, "PENDING")
		if err := svc.ApprovePlan(creator, plan.ID); err == nil {
			t.Fatal("creator approve must fail")
		}
		if err := svc.RejectPlan(creator, plan.ID); err == nil {
			t.Fatal("creator reject must fail")
		}
	})

	t.Run("expired pending plan cannot be approved", func(t *testing.T) {
		plan, token := buildOpenPlan(t, svc, "PENDING")
		past := time.Now().Add(-time.Minute)
		model.DB.Model(&model.ChangePlan{}).Where("id = ?", plan.ID).Update("expires_at", past)
		if err := svc.ApprovePlan(approverUser(), plan.ID); err == nil {
			t.Fatal("approving expired plan must fail")
		}
		var stored model.ChangePlan
		model.DB.First(&stored, plan.ID)
		if stored.Status != model.ChangePlanExpired {
			t.Fatalf("status = %s, want EXPIRED", stored.Status)
		}
		if _, err := svc.ApplyPlan(context.Background(), creator, plan.ID, token); err == nil {
			t.Fatal("apply on expired plan must fail")
		}
	})

	t.Run("finalize requires steps", func(t *testing.T) {
		plan, _, _ := svc.CreatePlan(creator, "", "", "empty")
		if err := svc.FinalizePlan(creator, plan.ID); err == nil {
			t.Fatal("finalize without steps must fail")
		}
	})

	t.Run("append after finalize fails", func(t *testing.T) {
		plan, _ := buildOpenPlan(t, svc, "PENDING")
		if _, err := svc.AppendStep(context.Background(), creator, plan.ID, "create_resource", "", map[string]any{"yaml": "x"}); err == nil {
			t.Fatal("append after finalize must fail")
		}
	})

	t.Run("only one concurrent finalize wins", func(t *testing.T) {
		user := creatorUser(7)
		plan, _, _ := svc.CreatePlan(user, "", "", "race")
		if _, err := svc.AppendStep(context.Background(), user, plan.ID, "create_resource", "", map[string]any{"yaml": "x"}); err != nil {
			t.Fatalf("AppendStep: %v", err)
		}
		var wg sync.WaitGroup
		var mu sync.Mutex
		wins := 0
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := svc.FinalizePlan(user, plan.ID)
				if err == nil {
					mu.Lock()
					wins++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if wins != 1 {
			t.Fatalf("finalize wins = %d, want exactly 1", wins)
		}
	})

	t.Run("append by non-owner fails", func(t *testing.T) {
		plan, _, _ := svc.CreatePlan(creator, "", "", "owned")
		other := creatorUser(42)
		if _, err := svc.AppendStep(context.Background(), other, plan.ID, "create_resource", "", map[string]any{"yaml": "x"}); err == nil {
			t.Fatal("non-owner append must fail")
		}
	})

	t.Run("non-mutation tool rejected in plans", func(t *testing.T) {
		plan, _, _ := svc.CreatePlan(creator, "", "", "d")
		if _, err := svc.AppendStep(context.Background(), creator, plan.ID, "get_pod_logs", "", map[string]any{}); err == nil {
			t.Fatal("read-only tool must be rejected as a plan step")
		}
	})

	t.Run("wrong token cannot apply", func(t *testing.T) {
		plan, token := buildOpenPlan(t, svc, "APPROVED")
		if _, err := svc.ApplyPlan(context.Background(), creator, plan.ID, "wrong-token"); err == nil {
			t.Fatal("wrong token must fail")
		}
		// The failed attempt must not burn the real token.
		svc.execute = func(context.Context, *cluster.ClientSet, model.User, string, map[string]any) (string, error) {
			return "ok", nil
		}
		if _, err := svc.ApplyPlan(context.Background(), creator, plan.ID, token); err != nil {
			t.Fatalf("correct token after failed attempt must still work: %v", err)
		}
	})
}

// TestPlanVisibility enforces that creators only see their plans unless they
// hold the approve permission.
func TestPlanVisibility(t *testing.T) {
	setupPlanDB(t)
	svc := newTestService()
	creator := creatorUser(1)
	plan, _, _ := svc.CreatePlan(creator, "", "", "secret")
	if _, err := svc.GetPlanForUser(plan.ID, creatorUser(2), false); err == nil {
		t.Fatal("other user without approve perm must not see the plan")
	}
	if _, err := svc.GetPlanForUser(plan.ID, creator, false); err != nil {
		t.Fatalf("creator must see own plan: %v", err)
	}
	if _, err := svc.GetPlanForUser(plan.ID, approverUser(), true); err != nil {
		t.Fatalf("approver must see the plan: %v", err)
	}

	others, err := svc.ListPlansForUser(creatorUser(2), false, model.ChangePlanCreated)
	if err != nil {
		t.Fatalf("ListPlansForUser: %v", err)
	}
	if len(others) != 0 {
		t.Fatalf("non-approver listing must not include other users' plans, got %d", len(others))
	}
}
