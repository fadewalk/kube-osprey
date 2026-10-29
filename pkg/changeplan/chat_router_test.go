package changeplan

import (
	"context"
	"testing"
	"time"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/model"
)

// TestChatRouterSubmitAndAutoApply drives the embedded-assistant flow:
// submit creates a kite-chat plan pending approval, and approving it applies
// the plan automatically without any token.
func TestChatRouterSubmitAndAutoApply(t *testing.T) {
	setupPlanDB(t)
	svc := newTestService()
	applied := make(chan string, 1)
	svc.execute = func(_ context.Context, _ *cluster.ClientSet, _ model.User, tool string, _ map[string]any) (string, error) {
		applied <- tool
		return "created", nil
	}
	router := svc.ChatRouter()
	creator := creatorUser(1)

	msg, err := router.SubmitMutation(context.Background(), &cluster.ClientSet{Name: "test"}, creator,
		"create_resource", map[string]any{"yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n"}, "sess-1")
	if err != nil {
		t.Fatalf("SubmitMutation: %v", err)
	}
	if msg == "" {
		t.Fatal("router must return a message for the model")
	}

	var plan model.ChangePlan
	if err := model.DB.Where("client_name = ?", ChatClientName).First(&plan).Error; err != nil {
		t.Fatalf("chat plan not found: %v", err)
	}
	if plan.Status != model.ChangePlanPending {
		t.Fatalf("status = %s, want PENDING", plan.Status)
	}
	if plan.AgentSessionID != "sess-1" {
		t.Fatalf("session id = %q", plan.AgentSessionID)
	}

	// Non-mutation tools are rejected and bury the plan as FAILED.
	if _, err := router.SubmitMutation(context.Background(), &cluster.ClientSet{Name: "test"}, creator,
		"get_pod_logs", map[string]any{}, "sess-2"); err == nil {
		t.Fatal("non-mutation submission must surface as an error")
	}
	var failed model.ChangePlan
	if err := model.DB.Where("client_name = ? AND status = ?", ChatClientName, model.ChangePlanFailed).
		First(&failed).Error; err != nil {
		t.Fatal("failed submission must leave a FAILED plan for audit")
	}

	// Approval triggers the auto-apply.
	if err := svc.ApprovePlan(approverUser(), plan.ID); err != nil {
		t.Fatalf("ApprovePlan: %v", err)
	}
	select {
	case tool := <-applied:
		if tool != "create_resource" {
			t.Fatalf("applied tool = %s", tool)
		}
	case <-timeAfter(2 * time.Second):
		t.Fatal("auto-apply did not run")
	}
	var appliedPlan model.ChangePlan
	for i := 0; i < 20; i++ {
		model.DB.First(&appliedPlan, plan.ID)
		if appliedPlan.Status == model.ChangePlanApplied {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if appliedPlan.Status != model.ChangePlanApplied {
		t.Fatalf("status = %s, want APPLIED", appliedPlan.Status)
	}
	if appliedPlan.ApprovalToken != "" {
		t.Fatal("token must be consumed")
	}
}

func timeAfter(d time.Duration) <-chan time.Time {
	return time.After(d)
}
