package ai

import (
	"context"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	pkgmodel "github.com/fadewalk/kube-osprey/pkg/model"
)

// ChangePlanRouter routes an AI mutation into the change plan approval flow
// instead of executing it directly. Implemented by the changeplan package to
// avoid an import cycle.
type ChangePlanRouter interface {
	// SubmitMutation creates a plan with the single mutation step, dry-runs
	// it and finalizes the plan for human approval. It returns a message for
	// the model; dry-run and validation failures come back as errors.
	SubmitMutation(ctx context.Context, cs *cluster.ClientSet, user pkgmodel.User, toolName string, args map[string]interface{}, sessionID string) (string, error)
}

var changePlanRouter ChangePlanRouter

// SetChangePlanRouter installs the router; called once at startup.
func SetChangePlanRouter(router ChangePlanRouter) {
	changePlanRouter = router
}

// changePlanModeEnabled reports whether chat mutations must go through the
// approval flow.
func changePlanModeEnabled() bool {
	setting, err := pkgmodel.GetGeneralSetting()
	if err != nil {
		return false
	}
	return setting.AIChangePlanMode
}

// ChangePlanRouterIfEnabled returns the router when plan mode is on, so the
// agent can bypass the in-chat confirmation for mutations.
func ChangePlanRouterIfEnabled() ChangePlanRouter {
	if changePlanRouter == nil || !changePlanModeEnabled() {
		return nil
	}
	return changePlanRouter
}
