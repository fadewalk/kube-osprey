package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zxh326/kite/pkg/ai"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// All Phase 1 tools are read-only; write operations go through the Phase 2
// change-plan approval flow instead of direct tools.
const dangerLevelReadOnly = "LOW"

const deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"

const (
	maxLogTailLines   = 1000
	logFetchTimeout   = 10 * time.Second
	defaultListLimit  = 50
	maxListLimit      = 200
	defaultTailLines  = 100
	prometheusTimeout = 15 * time.Second
)

type toolPermission struct {
	Resource  string
	Verb      string
	Namespace string
}

type toolContext struct {
	cm   *cluster.ClusterManager
	user model.User
	cs   *cluster.ClientSet
}

type toolDef struct {
	name        string
	description string
	schema      string
	// resolveCluster resolves args["cluster"] (default cluster when empty)
	// into toolContext.cs before permissions and execution.
	resolveCluster bool
	// readOnly marks the tool as non-mutating for MCP annotations and the
	// default LOW danger level of its result envelope.
	readOnly bool
	// danger overrides the envelope danger level for non-read-only tools.
	danger string
	// permissions returns the RBAC checks for this call; an empty list means
	// every authenticated user may run it (delegated tools enforce access in
	// ai.RunTool instead).
	permissions func(ctx context.Context, tctx *toolContext, args map[string]any) ([]toolPermission, error)
	run         func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error)
}

func registerTools(server *mcp.Server, cm *cluster.ClusterManager) {
	for _, def := range append(toolDefs(), planTools(cm)...) {
		server.AddTool(&mcp.Tool{
			Name:        def.name,
			Description: def.description,
			InputSchema: json.RawMessage(def.schema),
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: def.readOnly},
		}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return dispatchTool(ctx, cm, def, req)
		})
	}
}

func dispatchTool(ctx context.Context, cm *cluster.ClusterManager, def toolDef, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	user, ok := userFromContext(ctx)
	if !ok {
		return toolError("authenticated user not found in request context"), nil
	}
	args, err := decodeArgs(req.Params.Arguments)
	if err != nil {
		return toolError(err.Error()), nil
	}

	tctx := &toolContext{cm: cm, user: user}
	if def.resolveCluster {
		cs, csErr := tctx.cm.GetClientSet(optionalString(args, "cluster"))
		if csErr != nil {
			return toolError(csErr.Error()), nil
		}
		tctx.cs = cs
	}

	start := time.Now()
	clusterName := clusterLabel(tctx, args)
	perms, err := def.permissions(ctx, tctx, args)
	if err == nil {
		err = authorizeToolCall(tctx, clusterName, perms)
	}
	if err == nil {
		var data any
		data, err = def.run(ctx, tctx, args)
		if err == nil {
			recordAudit(user, clusterName, def.name, args, start, true, "")
			return toolResultFor(def, data), nil
		}
	}
	recordAudit(user, clusterName, def.name, args, start, false, err.Error())
	return toolError(err.Error()), nil
}

// authorizeToolCall evaluates the tool's permission list against the caller's
// roles; the first unmet permission aborts the call.
func authorizeToolCall(tctx *toolContext, clusterName string, perms []toolPermission) error {
	for _, p := range perms {
		if rbac.CanAccess(tctx.user, p.Resource, p.Verb, clusterName, p.Namespace) {
			continue
		}
		return fmt.Errorf(
			"forbidden: %s",
			rbac.NoAccess(tctx.user.Key(), p.Verb, p.Resource, p.Namespace, clusterName),
		)
	}
	return nil
}

func clusterLabel(tctx *toolContext, args map[string]any) string {
	if tctx.cs != nil {
		return tctx.cs.Name
	}
	return optionalString(args, "cluster")
}

type toolPayload struct {
	DangerLevel string `json:"dangerLevel"`
	Data        any    `json:"data"`
}

func toolResultFor(def toolDef, data any) *mcp.CallToolResult {
	danger := def.danger
	if def.readOnly || danger == "" {
		danger = dangerLevelReadOnly
	}
	payload := toolPayload{DangerLevel: danger, Data: data}
	b, err := json.Marshal(payload)
	if err != nil {
		return toolError(fmt.Sprintf("failed to encode result: %v", err))
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

func recordAudit(user model.User, clusterName, tool string, args map[string]any, start time.Time, success bool, errMsg string) {
	summary := ""
	if b, err := json.Marshal(args); err == nil {
		summary = truncateString(string(b), 512)
	}
	model.RecordMCPAudit(model.MCPAuditLog{
		UserID:       user.ID,
		Username:     user.Username,
		ToolName:     tool,
		ClusterName:  clusterName,
		ArgsSummary:  summary,
		DurationMS:   time.Since(start).Milliseconds(),
		Success:      success,
		ErrorMessage: errMsg,
	})
}

func getPermission(args map[string]any, resource string) []toolPermission {
	return []toolPermission{{
		Resource:  resource,
		Verb:      string(common.VerbGet),
		Namespace: permissionNamespace(args),
	}}
}

func workloadKinds(filter string) []string {
	switch filter {
	case "deployments", "deployment", "deploy":
		return []string{string(common.Deployments)}
	case "statefulsets", "statefulset", "sts":
		return []string{string(common.StatefulSets)}
	case "daemonsets", "daemonset", "ds":
		return []string{string(common.DaemonSets)}
	default:
		return []string{
			string(common.Deployments),
			string(common.StatefulSets),
			string(common.DaemonSets),
		}
	}
}

func requireWorkloadKind(args map[string]any) (string, error) {
	kind, err := requiredString(args, "kind")
	if err != nil {
		return "", err
	}
	normalized := kindOf(kind)
	if normalized != string(common.Deployments) &&
		normalized != string(common.StatefulSets) &&
		normalized != string(common.DaemonSets) {
		return "", fmt.Errorf("kind must be one of deployments, statefulsets, daemonsets")
	}
	return normalized, nil
}

func kindOf(kind string) string {
	switch kind {
	case "deployments", "deployment", "deploy":
		return string(common.Deployments)
	case "statefulsets", "statefulset", "sts":
		return string(common.StatefulSets)
	case "daemonsets", "daemonset", "ds":
		return string(common.DaemonSets)
	}
	return kind
}

func imagesOf(spec corev1.PodSpec) []string {
	images := make([]string, 0, len(spec.Containers))
	for _, c := range spec.Containers {
		images = append(images, c.Image)
	}
	return images
}

func selectorOf(obj client.Object) map[string]string {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		return v.Spec.Selector.MatchLabels
	case *appsv1.StatefulSet:
		return v.Spec.Selector.MatchLabels
	case *appsv1.DaemonSet:
		return v.Spec.Selector.MatchLabels
	}
	return nil
}

func listWorkloadObjects(ctx context.Context, tctx *toolContext, kind, namespace string) ([]client.Object, error) {
	switch kind {
	case string(common.Deployments):
		list := &appsv1.DeploymentList{}
		if err := tctx.cs.K8sClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
			return nil, err
		}
		objs := make([]client.Object, 0, len(list.Items))
		for i := range list.Items {
			objs = append(objs, &list.Items[i])
		}
		return objs, nil
	case string(common.StatefulSets):
		list := &appsv1.StatefulSetList{}
		if err := tctx.cs.K8sClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
			return nil, err
		}
		objs := make([]client.Object, 0, len(list.Items))
		for i := range list.Items {
			objs = append(objs, &list.Items[i])
		}
		return objs, nil
	case string(common.DaemonSets):
		list := &appsv1.DaemonSetList{}
		if err := tctx.cs.K8sClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
			return nil, err
		}
		objs := make([]client.Object, 0, len(list.Items))
		for i := range list.Items {
			objs = append(objs, &list.Items[i])
		}
		return objs, nil
	}
	return nil, fmt.Errorf("unsupported workload kind %q", kind)
}

func replicasOf(obj client.Object) (desired int32, ready int32, available int32) {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		if v.Spec.Replicas != nil {
			desired = *v.Spec.Replicas
		}
		return desired, v.Status.ReadyReplicas, v.Status.AvailableReplicas
	case *appsv1.StatefulSet:
		if v.Spec.Replicas != nil {
			desired = *v.Spec.Replicas
		}
		return desired, v.Status.ReadyReplicas, v.Status.AvailableReplicas
	case *appsv1.DaemonSet:
		return v.Status.DesiredNumberScheduled, v.Status.NumberReady, v.Status.NumberAvailable
	}
	return 0, 0, 0
}

func podSpecOf(obj client.Object) corev1.PodSpec {
	switch v := obj.(type) {
	case *appsv1.Deployment:
		return v.Spec.Template.Spec
	case *appsv1.StatefulSet:
		return v.Spec.Template.Spec
	case *appsv1.DaemonSet:
		return v.Spec.Template.Spec
	}
	return corev1.PodSpec{}
}

func coreResourceTools() []toolDef {
	return []toolDef{
		{
			name:        "list_clusters",
			description: "List clusters the current user can access, with the default cluster flagged.",
			schema:      `{"type":"object","properties":{}}`,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				// Access is filtered per cluster inside run.
				return nil, nil
			},
			run: func(_ context.Context, tctx *toolContext, _ map[string]any) (any, error) {
				clusters, errs, defaultContext := tctx.cm.SnapshotState()
				result := make([]common.ClusterInfo, 0, len(clusters))
				for name, cs := range clusters {
					if !rbac.CanAccessCluster(tctx.user, name) {
						continue
					}
					result = append(result, common.ClusterInfo{
						Name:      name,
						Version:   cs.Version,
						IsDefault: name == defaultContext,
					})
				}
				for name, errMsg := range errs {
					if !rbac.CanAccessCluster(tctx.user, name) {
						continue
					}
					result = append(result, common.ClusterInfo{Name: name, Error: errMsg})
				}
				sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
				return result, nil
			},
		},
		{
			name:        "list_namespaces",
			description: "List namespaces in a cluster.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."}},` +
				`"required":[]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return []toolPermission{{
					Resource: string(common.Namespaces),
					Verb:     string(common.VerbGet),
				}}, nil
			},
			run: func(ctx context.Context, tctx *toolContext, _ map[string]any) (any, error) {
				list, err := tctx.cs.K8sClient.ClientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}
				namespaces := make([]map[string]string, 0, len(list.Items))
				for _, ns := range list.Items {
					namespaces = append(namespaces, map[string]string{
						"name":   ns.Name,
						"status": string(ns.Status.Phase),
					})
				}
				sort.Slice(namespaces, func(i, j int) bool {
					return namespaces[i]["name"] < namespaces[j]["name"]
				})
				return namespaces, nil
			},
		},
		{
			name:        "list_resources",
			description: "List Kubernetes resources of one kind and return a compact diagnostic summary. Use get_resource when full YAML is needed.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"kind":{"type":"string","description":"Resource kind or plural, e.g. Pod, pods, Deployment, widgets.example.com."},` +
				`"namespace":{"type":"string","description":"Namespace. Omit or use _all for all namespaces."}},` +
				`"required":["kind"]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				// Authorized inside ai.RunTool with the same permission rules.
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				return ai.RunTool(ctx, tctx.cs, tctx.user, "list_resources", args)
			},
		},
		{
			name:        "get_resource",
			description: "Get one Kubernetes resource as YAML. Pass kind, name, and namespace as separate fields.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"kind":{"type":"string","description":"Resource kind or plural."},` +
				`"name":{"type":"string","description":"Resource name without namespace prefix."},` +
				`"namespace":{"type":"string","description":"Namespace for namespaced resources. Omit for cluster-scoped."}},` +
				`"required":["kind","name"]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				return ai.RunTool(ctx, tctx.cs, tctx.user, "get_resource", args)
			},
		},
	}
}

func workloadTools() []toolDef {
	return []toolDef{
		{
			name:        "list_workloads",
			description: "List workloads (deployments, statefulsets, daemonsets) with replica status and images.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"namespace":{"type":"string","description":"Namespace. Omit or use _all for all namespaces."},` +
				`"kind":{"type":"string","enum":["deployments","statefulsets","daemonsets"],"description":"Restrict to one workload kind. Omit for all."}},` +
				`"required":[]}`,
			resolveCluster: true,
			permissions: func(_ context.Context, _ *toolContext, args map[string]any) ([]toolPermission, error) {
				kinds := workloadKinds(optionalString(args, "kind"))
				perms := make([]toolPermission, 0, len(kinds))
				for _, kind := range kinds {
					perms = append(perms, toolPermission{
						Resource:  kind,
						Verb:      string(common.VerbGet),
						Namespace: permissionNamespace(args),
					})
				}
				return perms, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				namespace := namespaceScope(args)
				var rows []map[string]any
				for _, kind := range workloadKinds(optionalString(args, "kind")) {
					objs, err := listWorkloadObjects(ctx, tctx, kind, namespace)
					if err != nil {
						return nil, err
					}
					for _, obj := range objs {
						desired, ready, available := replicasOf(obj)
						rows = append(rows, map[string]any{
							"kind":      kind,
							"name":      obj.GetName(),
							"namespace": obj.GetNamespace(),
							"replicas": map[string]int32{
								"desired":   desired,
								"ready":     ready,
								"available": available,
							},
							"images": imagesOf(podSpecOf(obj)),
						})
					}
				}
				sort.Slice(rows, func(i, j int) bool {
					if rows[i]["namespace"] != rows[j]["namespace"] {
						return fmt.Sprint(rows[i]["namespace"]) < fmt.Sprint(rows[j]["namespace"])
					}
					return fmt.Sprint(rows[i]["name"]) < fmt.Sprint(rows[j]["name"])
				})
				return rows, nil
			},
		},
		{
			name:        "get_workload",
			description: "Get one workload (deployment, statefulset, or daemonset) with replica status, selector, and images.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"kind":{"type":"string","enum":["deployments","statefulsets","daemonsets"]},` +
				`"name":{"type":"string"},` +
				`"namespace":{"type":"string"}},` +
				`"required":["kind","name","namespace"]}`,
			resolveCluster: true,
			permissions: func(_ context.Context, _ *toolContext, args map[string]any) ([]toolPermission, error) {
				kind, err := requireWorkloadKind(args)
				if err != nil {
					return nil, err
				}
				return []toolPermission{{
					Resource:  kind,
					Verb:      string(common.VerbGet),
					Namespace: optionalString(args, "namespace"),
				}}, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				kind, err := requireWorkloadKind(args)
				if err != nil {
					return nil, err
				}
				name, err := requiredString(args, "name")
				if err != nil {
					return nil, err
				}
				namespace, err := requiredString(args, "namespace")
				if err != nil {
					return nil, err
				}
				objs, err := listWorkloadObjects(ctx, tctx, kind, namespace)
				if err != nil {
					return nil, err
				}
				for _, obj := range objs {
					if obj.GetName() != name {
						continue
					}
					desired, ready, available := replicasOf(obj)
					return map[string]any{
						"kind":      kind,
						"name":      obj.GetName(),
						"namespace": obj.GetNamespace(),
						"replicas": map[string]int32{
							"desired":   desired,
							"ready":     ready,
							"available": available,
						},
						"selector": selectorOf(obj),
						"images":   imagesOf(podSpecOf(obj)),
						"age":      time.Since(obj.GetCreationTimestamp().Time).Round(time.Second).String(),
					}, nil
				}
				return nil, fmt.Errorf("workload %s/%s not found", namespace, name)
			},
		},
		{
			name:        "get_workload_history",
			description: "List the ReplicaSets owned by a deployment, newest revision first, to inspect rollout history.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"name":{"type":"string","description":"Deployment name."},` +
				`"namespace":{"type":"string"}},` +
				`"required":["name","namespace"]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				// ReplicaSet visibility follows Deployment visibility.
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				name, err := requiredString(args, "name")
				if err != nil {
					return nil, err
				}
				namespace, err := requiredString(args, "namespace")
				if err != nil {
					return nil, err
				}
				if !rbac.CanAccess(tctx.user, string(common.Deployments), string(common.VerbGet), tctx.cs.Name, namespace) {
					return nil, fmt.Errorf("forbidden: %s", rbac.NoAccess(tctx.user.Key(), string(common.VerbGet), string(common.Deployments), namespace, tctx.cs.Name))
				}
				list := &appsv1.ReplicaSetList{}
				if err := tctx.cs.K8sClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
					return nil, err
				}
				owned := make([]appsv1.ReplicaSet, 0, len(list.Items))
				for _, rs := range list.Items {
					for _, ref := range rs.OwnerReferences {
						if ref.Name == name {
							owned = append(owned, rs)
							break
						}
					}
				}
				sort.Slice(owned, func(i, j int) bool {
					ri, _ := strconv.Atoi(owned[i].Annotations[deploymentRevisionAnnotation])
					rj, _ := strconv.Atoi(owned[j].Annotations[deploymentRevisionAnnotation])
					return ri > rj
				})
				revisions := make([]map[string]any, 0, len(owned))
				for _, rs := range owned {
					revision, _ := strconv.Atoi(rs.Annotations[deploymentRevisionAnnotation])
					revisions = append(revisions, map[string]any{
						"name":     rs.Name,
						"revision": revision,
						"replicas": map[string]int32{
							"desired": int32(1),
							"ready":   rs.Status.ReadyReplicas,
						},
						"images": imagesOf(rs.Spec.Template.Spec),
						"age":    time.Since(rs.CreationTimestamp.Time).Round(time.Second).String(),
					})
				}
				for i := range revisions {
					if d := owned[i].Spec.Replicas; d != nil {
						revisions[i]["replicas"].(map[string]int32)["desired"] = *d
					}
				}
				return revisions, nil
			},
		},
	}
}

func podTools() []toolDef {
	return []toolDef{
		{
			name:        "get_pod",
			description: "Get one pod with phase, node, addresses, container states, and restart counts.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"name":{"type":"string"},` +
				`"namespace":{"type":"string"}},` +
				`"required":["name","namespace"]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				name, err := requiredString(args, "name")
				if err != nil {
					return nil, err
				}
				namespace, err := requiredString(args, "namespace")
				if err != nil {
					return nil, err
				}
				pod := &corev1.Pod{}
				if err := tctx.cs.K8sClient.Get(ctx, k8stypes.NamespacedName{Namespace: namespace, Name: name}, pod); err != nil {
					return nil, err
				}
				containers := make([]map[string]any, 0, len(pod.Status.ContainerStatuses))
				for _, cs := range pod.Status.ContainerStatuses {
					state := "waiting"
					reason := ""
					switch {
					case cs.State.Running != nil:
						state = "running"
					case cs.State.Terminated != nil:
						state = "terminated"
						reason = cs.State.Terminated.Reason
					case cs.State.Waiting != nil:
						reason = cs.State.Waiting.Reason
					}
					containers = append(containers, map[string]any{
						"name":         cs.Name,
						"image":        cs.Image,
						"ready":        cs.Ready,
						"restartCount": cs.RestartCount,
						"state":        state,
						"reason":       reason,
					})
				}
				summary := map[string]any{
					"name":       pod.Name,
					"namespace":  pod.Namespace,
					"phase":      pod.Status.Phase,
					"node":       pod.Spec.NodeName,
					"podIP":      pod.Status.PodIP,
					"hostIP":     pod.Status.HostIP,
					"containers": containers,
				}
				if pod.Status.StartTime != nil {
					summary["startedAt"] = pod.Status.StartTime.Time.UTC().Format(time.RFC3339)
				}
				return summary, nil
			},
		},
		{
			name:        "get_pod_logs",
			description: "Fetch recent logs of one pod container. tail_lines is capped at 1000.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"name":{"type":"string"},` +
				`"namespace":{"type":"string"},` +
				`"container":{"type":"string","description":"Container name. Omit for the first container."},` +
				`"tail_lines":{"type":"integer","description":"Lines to fetch, default 100, max 1000."},` +
				`"previous":{"type":"boolean","description":"Logs of the previous container instance."}},` +
				`"required":["name","namespace"]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				tail := optionalInt(args, "tail_lines", defaultTailLines)
				if tail <= 0 {
					tail = defaultTailLines
				}
				if tail > maxLogTailLines {
					tail = maxLogTailLines
				}
				args["tail_lines"] = float64(tail)
				logCtx, cancel := context.WithTimeout(ctx, logFetchTimeout)
				defer cancel()
				return ai.RunTool(logCtx, tctx.cs, tctx.user, "get_pod_logs", args)
			},
		},
	}
}

func accessTools() []toolDef {
	return []toolDef{
		{
			name:        "list_events",
			description: "List recent events of a namespace or the whole cluster, most recent first.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"namespace":{"type":"string","description":"Namespace. Omit or use _all for all namespaces."},` +
				`"limit":{"type":"integer","description":"Max events, default 50, max 200."}},` +
				`"required":[]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return getPermission(map[string]any{}, string(common.Events)), nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				limit := optionalInt(args, "limit", defaultListLimit)
				if limit <= 0 {
					limit = defaultListLimit
				}
				if limit > maxListLimit {
					limit = maxListLimit
				}
				list := &corev1.EventList{}
				if err := tctx.cs.K8sClient.List(ctx, list, client.InNamespace(namespaceScope(args))); err != nil {
					return nil, err
				}
				items := list.Items
				sort.Slice(items, func(i, j int) bool {
					ti, tj := items[i].LastTimestamp, items[j].LastTimestamp
					if ti.IsZero() {
						ti = items[i].FirstTimestamp
					}
					if tj.IsZero() {
						tj = items[j].FirstTimestamp
					}
					return ti.After(tj.Time)
				})
				if len(items) > limit {
					items = items[:limit]
				}
				events := make([]map[string]any, 0, len(items))
				for _, ev := range items {
					last := ev.LastTimestamp
					if last.IsZero() {
						last = ev.FirstTimestamp
					}
					entry := map[string]any{
						"type":     ev.Type,
						"reason":   ev.Reason,
						"object":   fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name),
						"message":  ev.Message,
						"count":    ev.Count,
						"lastSeen": last.Time.UTC().Format(time.RFC3339),
					}
					if ev.InvolvedObject.Namespace != "" {
						entry["namespace"] = ev.InvolvedObject.Namespace
					}
					events = append(events, entry)
				}
				return events, nil
			},
		},
		{
			name:        "list_endpoints",
			description: "List service endpoints with ready addresses and ports.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"namespace":{"type":"string","description":"Namespace. Omit or use _all for all namespaces."}},` +
				`"required":[]}`,
			resolveCluster: true,
			permissions: func(_ context.Context, _ *toolContext, args map[string]any) ([]toolPermission, error) {
				ns := permissionNamespace(args)
				return []toolPermission{
					{Resource: string(common.Services), Verb: string(common.VerbGet), Namespace: ns},
					{Resource: string(common.Endpoints), Verb: string(common.VerbGet), Namespace: ns},
				}, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				list := &corev1.EndpointsList{}
				if err := tctx.cs.K8sClient.List(ctx, list, client.InNamespace(namespaceScope(args))); err != nil {
					return nil, err
				}
				endpoints := make([]map[string]any, 0, len(list.Items))
				for _, ep := range list.Items {
					subsets := make([]map[string]any, 0, len(ep.Subsets))
					for _, subset := range ep.Subsets {
						ports := make([]map[string]any, 0, len(subset.Ports))
						for _, port := range subset.Ports {
							ports = append(ports, map[string]any{
								"name":     port.Name,
								"port":     port.Port,
								"protocol": string(port.Protocol),
							})
						}
						subsets = append(subsets, map[string]any{
							"readyAddresses":    len(subset.Addresses),
							"notReadyAddresses": len(subset.NotReadyAddresses),
							"ports":             ports,
						})
					}
					endpoints = append(endpoints, map[string]any{
						"name":      ep.Name,
						"namespace": ep.Namespace,
						"subsets":   subsets,
					})
				}
				return endpoints, nil
			},
		},
		{
			name:        "check_permission",
			description: "Check whether the current user may perform a verb on a resource, evaluated against Kite RBAC.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"verb":{"type":"string","description":"Verb such as get, list, create, update, delete."},` +
				`"resource":{"type":"string","description":"Resource plural, e.g. pods, deployments."},` +
				`"namespace":{"type":"string","description":"Namespace scope. Omit for all namespaces."}},` +
				`"required":["verb","resource"]}`,
			// Kite RBAC is application-level; evaluating it must not require a
			// connected cluster.
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(_ context.Context, tctx *toolContext, args map[string]any) (any, error) {
				verb, err := requiredString(args, "verb")
				if err != nil {
					return nil, err
				}
				resource, err := requiredString(args, "resource")
				if err != nil {
					return nil, err
				}
				clusterName := clusterLabel(tctx, args)
				allowed := rbac.CanAccess(tctx.user, resource, verb, clusterName, permissionNamespace(args))
				return map[string]any{
					"allowed":   allowed,
					"verb":      verb,
					"resource":  resource,
					"namespace": permissionNamespace(args),
					"cluster":   clusterName,
				}, nil
			},
		},
	}
}

func metricsTools() []toolDef {
	return []toolDef{
		{
			name:        "get_node_metrics",
			description: "Get current node CPU and memory usage from metrics-server.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."}},` +
				`"required":[]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return []toolPermission{{
					Resource: string(common.Nodes),
					Verb:     string(common.VerbGet),
				}}, nil
			},
			run: func(ctx context.Context, tctx *toolContext, _ map[string]any) (any, error) {
				if tctx.cs.K8sClient.MetricsClient == nil {
					return nil, fmt.Errorf("metrics-server is not available for cluster %s", tctx.cs.Name)
				}
				list, err := tctx.cs.K8sClient.MetricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
				if err != nil {
					return nil, err
				}
				nodes := make([]map[string]any, 0, len(list.Items))
				for _, nm := range list.Items {
					nodes = append(nodes, map[string]any{
						"name":        nm.Name,
						"cpuMillis":   nm.Usage.Cpu().MilliValue(),
						"memoryBytes": nm.Usage.Memory().Value(),
					})
				}
				sort.Slice(nodes, func(i, j int) bool {
					return fmt.Sprint(nodes[i]["name"]) < fmt.Sprint(nodes[j]["name"])
				})
				return nodes, nil
			},
		},
		{
			name:        "get_pod_metrics",
			description: "Get current CPU and memory usage of one pod from metrics-server.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"name":{"type":"string"},` +
				`"namespace":{"type":"string"}},` +
				`"required":["name","namespace"]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				if tctx.cs.K8sClient.MetricsClient == nil {
					return nil, fmt.Errorf("metrics-server is not available for cluster %s", tctx.cs.Name)
				}
				name, err := requiredString(args, "name")
				if err != nil {
					return nil, err
				}
				namespace, err := requiredString(args, "namespace")
				if err != nil {
					return nil, err
				}
				pm, err := tctx.cs.K8sClient.MetricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					return nil, err
				}
				containers := make([]map[string]any, 0, len(pm.Containers))
				for _, c := range pm.Containers {
					containers = append(containers, map[string]any{
						"name":        c.Name,
						"cpuMillis":   c.Usage.Cpu().MilliValue(),
						"memoryBytes": c.Usage.Memory().Value(),
					})
				}
				return map[string]any{
					"name":       pm.Name,
					"namespace":  pm.Namespace,
					"timestamp":  pm.Timestamp.UTC().Format(time.RFC3339),
					"containers": containers,
				}, nil
			},
		},
		{
			name:        "prometheus_query",
			description: "Run an instant Prometheus (PromQL) query. Requires cluster-wide pod read access.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"query":{"type":"string","description":"PromQL query, e.g. sum(rate(container_cpu_usage_seconds_total[5m]))"}},` +
				`"required":["query"]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				if tctx.cs.PromClient == nil {
					return nil, fmt.Errorf("no prometheus client configured for cluster %s", tctx.cs.Name)
				}
				queryCtx, cancel := context.WithTimeout(ctx, prometheusTimeout)
				defer cancel()
				return ai.RunTool(queryCtx, tctx.cs, tctx.user, "query_prometheus", map[string]any{
					"query":      args["query"],
					"query_type": "instant",
				})
			},
		},
		{
			name:        "prometheus_query_range",
			description: "Run a Prometheus range query over a duration such as 30m, 6h, or 7d.",
			schema: `{"type":"object","properties":{` +
				`"cluster":{"type":"string","description":"Cluster name. Omit for the default cluster."},` +
				`"query":{"type":"string","description":"PromQL query."},` +
				`"duration":{"type":"string","description":"Lookback window, default 1h."}},` +
				`"required":["query"]}`,
			resolveCluster: true,
			permissions: func(context.Context, *toolContext, map[string]any) ([]toolPermission, error) {
				return nil, nil
			},
			run: func(ctx context.Context, tctx *toolContext, args map[string]any) (any, error) {
				if tctx.cs.PromClient == nil {
					return nil, fmt.Errorf("no prometheus client configured for cluster %s", tctx.cs.Name)
				}
				duration := optionalString(args, "duration")
				if duration == "" {
					duration = "1h"
				}
				queryCtx, cancel := context.WithTimeout(ctx, prometheusTimeout)
				defer cancel()
				return ai.RunTool(queryCtx, tctx.cs, tctx.user, "query_prometheus", map[string]any{
					"query":      args["query"],
					"query_type": "range",
					"duration":   duration,
				})
			},
		},
	}
}

// toolDefs aggregates every read-only tool exposed on the MCP endpoint.
func toolDefs() []toolDef {
	var defs []toolDef
	defs = append(defs, coreResourceTools()...)
	defs = append(defs, workloadTools()...)
	defs = append(defs, podTools()...)
	defs = append(defs, accessTools()...)
	defs = append(defs, metricsTools()...)
	return defs
}
