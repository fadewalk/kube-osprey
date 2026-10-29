package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/common"
	pkgmodel "github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/fadewalk/kube-osprey/pkg/rbac"
	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

type toolPermission struct {
	Resource  string
	Verb      string
	Namespace string
}

func permissionNamespace(resource resourceInfo, namespace string) string {
	if resource.ClusterScoped {
		return ""
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return common.AllNamespaces
	}
	return namespace
}

func requiredToolPermissions(ctx context.Context, cs *cluster.ClientSet, toolName string, args map[string]interface{}) ([]toolPermission, error) {
	switch toolName {
	case "get_resource", "describe_resource":
		kind, err := getRequiredString(args, "kind")
		if err != nil {
			return nil, err
		}
		namespace, _ := args["namespace"].(string)
		resource := resolveResourceInfo(ctx, cs, kind)
		return []toolPermission{{
			Resource:  common.HistoryResourceType(resource.Resource, resource.Group),
			Verb:      string(common.VerbGet),
			Namespace: permissionNamespace(resource, namespace),
		}}, nil
	case "list_resources":
		kind, err := getRequiredString(args, "kind")
		if err != nil {
			return nil, err
		}
		namespace, _ := args["namespace"].(string)
		resource := resolveResourceInfo(ctx, cs, kind)
		return []toolPermission{{
			Resource:  common.HistoryResourceType(resource.Resource, resource.Group),
			Verb:      string(common.VerbGet),
			Namespace: permissionNamespace(resource, namespace),
		}}, nil
	case "get_pod_logs":
		if _, err := getRequiredString(args, "name"); err != nil {
			return nil, err
		}
		namespace, err := getRequiredString(args, "namespace")
		if err != nil {
			return nil, err
		}
		return []toolPermission{{
			Resource:  string(common.Pods),
			Verb:      string(common.VerbLog),
			Namespace: namespace,
		}}, nil
	case "exec_in_pod":
		options, err := parseExecInPodOptions(args)
		if err != nil {
			return nil, err
		}
		return []toolPermission{{
			Resource:  string(common.Pods),
			Verb:      string(common.VerbExec),
			Namespace: options.Namespace,
		}}, nil
	case "get_cluster_overview":
		return []toolPermission{
			{Resource: string(common.Nodes), Verb: string(common.VerbGet), Namespace: ""},
			{Resource: string(common.Pods), Verb: string(common.VerbGet), Namespace: common.AllNamespaces},
			{Resource: string(common.Namespaces), Verb: string(common.VerbGet), Namespace: ""},
			{Resource: string(common.Services), Verb: string(common.VerbGet), Namespace: common.AllNamespaces},
		}, nil
	case "create_resource":
		obj, err := parseResourceYAML(args)
		if err != nil {
			return nil, err
		}
		resource := resolveResourceInfoForObject(ctx, cs, obj)
		return []toolPermission{{
			Resource:  common.HistoryResourceType(resource.Resource, resource.Group),
			Verb:      string(common.VerbCreate),
			Namespace: permissionNamespace(resource, obj.GetNamespace()),
		}}, nil
	case "apply_resource":
		obj, err := parseResourceYAML(args)
		if err != nil {
			return nil, err
		}
		resource := resolveResourceInfoForObject(ctx, cs, obj)
		existing := buildObjectForResource(resource)
		err = cs.K8sClient.Get(ctx, k8stypes.NamespacedName{
			Name:      obj.GetName(),
			Namespace: normalizeNamespace(resource, obj.GetNamespace()),
		}, existing)
		verb := common.VerbUpdate
		if apierrors.IsNotFound(err) {
			verb = common.VerbCreate
		} else if err != nil {
			return nil, err
		}
		return []toolPermission{{
			Resource:  common.HistoryResourceType(resource.Resource, resource.Group),
			Verb:      string(verb),
			Namespace: permissionNamespace(resource, obj.GetNamespace()),
		}}, nil
	case "update_resource":
		obj, err := parseResourceYAML(args)
		if err != nil {
			return nil, err
		}
		resource := resolveResourceInfoForObject(ctx, cs, obj)
		return []toolPermission{{
			Resource:  common.HistoryResourceType(resource.Resource, resource.Group),
			Verb:      string(common.VerbUpdate),
			Namespace: permissionNamespace(resource, obj.GetNamespace()),
		}}, nil
	case "patch_resource":
		kind, err := getRequiredString(args, "kind")
		if err != nil {
			return nil, err
		}
		if _, err := getRequiredString(args, "name"); err != nil {
			return nil, err
		}
		namespace, _ := args["namespace"].(string)
		resource := resolveResourceInfo(ctx, cs, kind)
		return []toolPermission{{
			Resource:  common.HistoryResourceType(resource.Resource, resource.Group),
			Verb:      string(common.VerbUpdate),
			Namespace: permissionNamespace(resource, namespace),
		}}, nil
	case "delete_resource":
		kind, err := getRequiredString(args, "kind")
		if err != nil {
			return nil, err
		}
		if _, err := getRequiredString(args, "name"); err != nil {
			return nil, err
		}
		namespace, _ := args["namespace"].(string)
		resource := resolveResourceInfo(ctx, cs, kind)
		return []toolPermission{{
			Resource:  common.HistoryResourceType(resource.Resource, resource.Group),
			Verb:      string(common.VerbDelete),
			Namespace: permissionNamespace(resource, namespace),
		}}, nil
	case "helm_upgrade_release":
		// A helm release upgrade writes release storage (secrets) in the
		// release namespace; require update on secrets there.
		namespace, err := getRequiredString(args, "namespace")
		if err != nil {
			return nil, err
		}
		if _, err := getRequiredString(args, "release"); err != nil {
			return nil, err
		}
		return []toolPermission{{
			Resource:  common.HistoryResourceType("secrets", ""),
			Verb:      string(common.VerbUpdate),
			Namespace: namespace,
		}}, nil
	case "query_prometheus":
		// Prometheus queries can access metrics from any namespace
		// Require at least read permission on pods in all namespaces
		// This ensures users can only query metrics if they have cluster-wide read access
		return []toolPermission{{
			Resource:  string(common.Pods),
			Verb:      string(common.VerbGet),
			Namespace: common.AllNamespaces,
		}}, nil
	default:
		return nil, nil
	}
}

func currentUserFromGin(c *gin.Context) (pkgmodel.User, bool) {
	rawUser, ok := c.Get("user")
	if !ok {
		return pkgmodel.User{}, false
	}
	user, ok := rawUser.(pkgmodel.User)
	return user, ok
}

// AuthorizeToolForUser authorizes a tool call made on behalf of an explicit
// user without a gin context, so non-HTTP surfaces (the MCP endpoint) share
// the same permission semantics as the chat assistant.
func AuthorizeToolForUser(ctx context.Context, cs *cluster.ClientSet, user pkgmodel.User, toolName string, args map[string]interface{}) error {
	if cs == nil {
		return errors.New("cluster client is required")
	}
	permissions, err := requiredToolPermissions(ctx, cs, toolName, args)
	if err != nil {
		return err
	}

	for _, permission := range permissions {
		if rbac.CanAccess(user, permission.Resource, permission.Verb, cs.Name, permission.Namespace) {
			continue
		}
		return fmt.Errorf("forbidden: %s", rbac.NoAccess(user.Key(), permission.Verb, permission.Resource, permission.Namespace, cs.Name))
	}
	return nil
}

func AuthorizeTool(c *gin.Context, cs *cluster.ClientSet, toolName string, args map[string]interface{}) (string, bool) {
	if c == nil {
		return "Error: authorization context is required", true
	}
	if cs == nil {
		return "Error: cluster client is required", true
	}
	user, ok := currentUserFromGin(c)
	if !ok {
		return "Error: authenticated user not found in context", true
	}
	if err := AuthorizeToolForUser(c.Request.Context(), cs, user, toolName, args); err != nil {
		return "Error: " + err.Error(), true
	}
	return "", false
}

// readOnlyTools lists built-in tools that may run without the change-plan
// approval gate. Write tools stay chat-only until the plan workflow lands.
var readOnlyTools = map[string]struct{}{
	"get_resource":         {},
	"describe_resource":    {},
	"list_resources":       {},
	"get_pod_logs":         {},
	"get_cluster_overview": {},
	"query_prometheus":     {},
}

// RunTool authorizes and executes a read-only built-in tool for an explicit
// user and returns its textual result. It backs the MCP tool surface.
func RunTool(ctx context.Context, cs *cluster.ClientSet, user pkgmodel.User, toolName string, args map[string]interface{}) (string, error) {
	if _, ok := readOnlyTools[toolName]; !ok {
		return "", fmt.Errorf("tool %q is not available on this surface", toolName)
	}
	if err := AuthorizeToolForUser(ctx, cs, user, toolName, args); err != nil {
		return "", err
	}
	result, isError := executeBuiltinTool(ctx, cs, user, toolName, args)
	if isError {
		return "", errors.New(result)
	}
	return result, nil
}

// planMutationTools lists the built-in write tools a change plan step may
// carry. Execution goes exclusively through ExecuteMutation, which re-checks
// RBAC at apply time; nothing else may trigger these against a cluster.
var planMutationTools = map[string]struct{}{
	"create_resource":      {},
	"apply_resource":       {},
	"update_resource":      {},
	"patch_resource":       {},
	"delete_resource":      {},
	"helm_upgrade_release": {},
}

// IsPlanMutationTool reports whether toolName is an approvable write tool.
func IsPlanMutationTool(toolName string) bool {
	_, ok := planMutationTools[toolName]
	return ok
}

// ExecuteMutation authorizes and executes a plan-approved write tool for the
// plan creator's user. It is the single execution path for change plan steps.
func ExecuteMutation(ctx context.Context, cs *cluster.ClientSet, user pkgmodel.User, toolName string, args map[string]interface{}) (string, error) {
	if _, ok := planMutationTools[toolName]; !ok {
		return "", fmt.Errorf("tool %q is not allowed in change plans", toolName)
	}
	if err := AuthorizeToolForUser(ctx, cs, user, toolName, args); err != nil {
		return "", err
	}
	result, isError := executeBuiltinTool(ctx, cs, user, toolName, args)
	if isError {
		return "", errors.New(result)
	}
	return result, nil
}

// ExecuteTool runs a tool and returns the result as a string.
func ExecuteTool(ctx context.Context, c *gin.Context, cs *cluster.ClientSet, toolName string, args map[string]interface{}) (string, bool) {
	if result, isError := AuthorizeTool(c, cs, toolName, args); isError {
		return result, true
	}

	user, _ := currentUserFromGin(c)

	// Plan mode: chat mutations become approval-gated change plans instead
	// of direct execution.
	if changePlanRouter != nil && IsPlanMutationTool(toolName) && changePlanModeEnabled() {
		msg, err := changePlanRouter.SubmitMutation(ctx, cs, user, toolName, args, c.GetString("ai_session_id"))
		if err != nil {
			return "Error: " + err.Error(), true
		}
		return msg, false
	}

	return executeBuiltinTool(ctx, cs, user, toolName, args)
}

func executeBuiltinTool(ctx context.Context, cs *cluster.ClientSet, user pkgmodel.User, toolName string, args map[string]interface{}) (string, bool) {
	switch toolName {
	case "get_resource":
		return executeGetResource(ctx, cs, args)
	case "describe_resource":
		return executeDescribeResource(ctx, cs, args)
	case "list_resources":
		return executeListResources(ctx, cs, args)
	case "get_pod_logs":
		return executeGetPodLogs(ctx, cs, args)
	case "exec_in_pod":
		return executeExecInPod(ctx, cs, user, args)
	case "get_cluster_overview":
		return executeGetClusterOverview(ctx, cs)
	case "create_resource":
		return executeCreateResource(ctx, cs, user, args)
	case "apply_resource":
		return executeApplyResource(ctx, cs, user, args)
	case "update_resource":
		return executeUpdateResource(ctx, cs, user, args)
	case "patch_resource":
		return executePatchResource(ctx, cs, user, args)
	case "delete_resource":
		return executeDeleteResource(ctx, cs, user, args)
	case "helm_upgrade_release":
		result, err := executeHelmUpgradeRelease(ctx, cs, user, args)
		if err != nil {
			return "Error: " + err.Error(), true
		}
		return result, false
	case "query_prometheus":
		return executeQueryPrometheus(ctx, cs, args)
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName), true
	}
}
