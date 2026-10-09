package mcpserver

import (
	"context"
	"fmt"
	"strconv"

	"github.com/zxh326/kite/pkg/changeplan"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
)

// planTools registers the change-plan workflow: agents author and dry-run
// plans here, humans approve in the UI, and only apply_change_plan executes
// writes (with a one-time approval token).
func planTools(cm *cluster.ClusterManager) []toolDef {
	svc := changeplan.NewService(cm)
	return []toolDef{
		{
			name:        "create_change_plan",
			description: "Open an empty change plan. Returns the plan id and a one-time approval token; the token is required by apply_change_plan and shown only here.",
			schema: `{"type":"object","properties":{` +
				`"description":{"type":"string","description":"What this plan is going to change and why."},` +
				`"client_name":{"type":"string","description":"MCP client name, e.g. claude-code."}},` +
				`"required":[]}`,
			danger: model.DangerLevelMedium,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(_ context.Context, tctx *toolContext, args map[string]any) (any, error) {
				plan, token, err := svc.CreatePlan(
					tctx.user,
					optionalString(args, "client_name"),
					"",
					optionalString(args, "description"),
				)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"plan_id":        plan.ID,
					"approval_token": token,
					"status":         plan.Status,
					"note":           "Append steps, then finalize. After a human approves, call apply_change_plan with this plan_id and approval_token.",
				}, nil
			},
		},
		{
			name:        "append_change_plan_step",
			description: "Append one mutation (create/apply/update/patch/delete resource) to an open plan. The step is validated against your RBAC and immediately dry-run server-side; pass the same arguments as the standalone mutation tools plus plan_id and tool.",
			schema: `{"type":"object","properties":{` +
				`"plan_id":{"type":"integer"},` +
				`"tool":{"type":"string","enum":["create_resource","apply_resource","update_resource","patch_resource","delete_resource"]},` +
				`"description":{"type":"string"},` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"yaml":{"type":"string","description":"Resource YAML for create/apply/update."},` +
				`"kind":{"type":"string"},` +
				`"name":{"type":"string"},` +
				`"namespace":{"type":"string"},` +
				`"patch":{"type":"string","description":"JSON patch body for patch_resource."},` +
				`"patch_type":{"type":"string","enum":["strategic","merge","json"]}},` +
				`"required":["plan_id","tool"]}`,
			danger: model.DangerLevelMedium,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				planID, err := planIDArg(args)
				if err != nil {
					return nil, err
				}
				tool, err := requiredString(args, "tool")
				if err != nil {
					return nil, err
				}
				mutationArgs := map[string]any{}
				for k, v := range args {
					switch k {
					case "plan_id", "tool", "description":
						continue
					}
					mutationArgs[k] = v
				}
				step, err := svc.AppendStep(ctx, tctx.user, planID, tool, optionalString(args, "description"), mutationArgs)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"seq":          step.Seq,
					"dry_run":      step.DryRunStatus,
					"danger_level": step.DangerLevel,
					"cluster":      step.Cluster,
					"before":       step.BeforeJSON,
					"after":        step.AfterJSON,
				}, nil
			},
		},
		{
			name:        "finalize_change_plan",
			description: "Seal an open plan and submit it for human approval. The plan becomes visible in the approval center and can no longer be modified.",
			schema: `{"type":"object","properties":{` +
				`"plan_id":{"type":"integer"}},` +
				`"required":["plan_id"]}`,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(_ context.Context, tctx *toolContext, args map[string]any) (any, error) {
				planID, err := planIDArg(args)
				if err != nil {
					return nil, err
				}
				if err := svc.FinalizePlan(tctx.user, planID); err != nil {
					return nil, err
				}
				return map[string]any{
					"plan_id": planID,
					"status":  model.ChangePlanPending,
					"note":    "Poll get_change_plan_status until the plan is approved or rejected, then apply with apply_change_plan.",
				}, nil
			},
		},
		{
			name:        "get_change_plan_status",
			description: "Read a plan's approval state and per-step execution status.",
			schema: `{"type":"object","properties":{` +
				`"plan_id":{"type":"integer"}},` +
				`"required":["plan_id"]}`,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(_ context.Context, tctx *toolContext, args map[string]any) (any, error) {
				planID, err := planIDArg(args)
				if err != nil {
					return nil, err
				}
				plan, err := svc.GetPlanForUser(planID, tctx.user, changeplan.CanApproveChangePlans(tctx.user))
				if err != nil {
					return nil, fmt.Errorf("plan %d not found", planID)
				}
				steps := make([]map[string]any, 0, len(plan.Steps))
				for _, s := range plan.Steps {
					steps = append(steps, map[string]any{
						"seq":       s.Seq,
						"tool":      s.ToolName,
						"cluster":   s.Cluster,
						"namespace": s.Namespace,
						"resource":  s.Resource,
						"status":    s.Status,
						"dry_run":   s.DryRunStatus,
						"apply":     s.ApplyStatus,
						"error":     s.ErrorMessage,
					})
				}
				return map[string]any{
					"plan_id":     plan.ID,
					"description": plan.Description,
					"status":      plan.Status,
					"danger":      plan.DangerLevel,
					"expires_at":  plan.ExpiresAt,
					"approver":    plan.ApproverName,
					"steps":       steps,
				}, nil
			},
		},
		{
			name:        "apply_change_plan",
			description: "Execute an approved plan with the one-time approval token from create_change_plan. Steps run in order and stop at the first failure; RBAC is re-checked per step at execution time.",
			schema: `{"type":"object","properties":{` +
				`"plan_id":{"type":"integer"},` +
				`"approval_token":{"type":"string"}},` +
				`"required":["plan_id","approval_token"]}`,
			danger: model.DangerLevelHigh,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				planID, err := planIDArg(args)
				if err != nil {
					return nil, err
				}
				token, err := requiredString(args, "approval_token")
				if err != nil {
					return nil, err
				}
				plan, applyErr := svc.ApplyPlan(ctx, tctx.user, planID, token)
				if plan == nil {
					return nil, applyErr
				}
				steps := make([]map[string]any, 0, len(plan.Steps))
				for _, s := range plan.Steps {
					steps = append(steps, map[string]any{
						"seq":     s.Seq,
						"tool":    s.ToolName,
						"status":  s.Status,
						"error":   s.ErrorMessage,
						"applied": s.ApplyStatus == model.ChangeStepApplied,
					})
				}
				result := map[string]any{
					"plan_id": planID,
					"status":  plan.Status,
					"steps":   steps,
				}
				if applyErr != nil {
					result["error"] = applyErr.Error()
				}
				return result, nil
			},
		},
	}
}

func planIDArg(args map[string]any) (uint, error) {
	switch v := args["plan_id"].(type) {
	case float64:
		if v <= 0 {
			return 0, fmt.Errorf("plan_id must be a positive integer")
		}
		return uint(v), nil
	case string:
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			return 0, fmt.Errorf("plan_id must be a positive integer")
		}
		return uint(n), nil
	}
	return 0, fmt.Errorf("plan_id is required")
}
