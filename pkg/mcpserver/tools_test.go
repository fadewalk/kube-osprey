package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/fadewalk/kube-osprey/pkg/model"
)

func toolDefByName(t *testing.T, name string) toolDef {
	t.Helper()
	for _, def := range toolDefs() {
		if def.name == name {
			return def
		}
	}
	t.Fatalf("tool %q not registered", name)
	return toolDef{}
}

func allAccessUser(name string) model.User {
	return model.User{
		Username: name,
		Roles: []common.Role{{
			Name:       "full-access",
			Clusters:   []string{"*"},
			Namespaces: []string{"*"},
			Resources:  []string{"*"},
			Verbs:      []string{"*"},
		}},
	}
}

func podsReadUser() model.User {
	return model.User{
		Username: "podsonly",
		Roles: []common.Role{{
			Name:       "pods-reader",
			Clusters:   []string{"*"},
			Namespaces: []string{"*"},
			Resources:  []string{"pods"},
			Verbs:      []string{"get"},
		}},
	}
}

func noRoleUser(name string) model.User {
	return model.User{Username: name}
}

// TestToolRBACDenialPaths drives the permission resolution of every registered
// tool as different users to pin down the denial behavior.
func TestToolRBACDenialPaths(t *testing.T) {
	tctx := &toolContext{
		user: noRoleUser("limited"),
		cs:   &cluster.ClientSet{Name: "test"},
		cm:   &cluster.ClusterManager{},
	}
	ctx := context.Background()

	tests := []struct {
		name    string
		tool    string
		args    map[string]any
		wantDen bool
	}{
		{name: "no role denied on list_namespaces", tool: "list_namespaces", args: map[string]any{}, wantDen: true},
		{name: "no role denied on list_workloads", tool: "list_workloads", args: map[string]any{}, wantDen: true},
		{name: "no role denied on list_events", tool: "list_events", args: map[string]any{}, wantDen: true},
		{name: "no role denied on list_endpoints", tool: "list_endpoints", args: map[string]any{}, wantDen: true},
		{name: "no role denied on get_node_metrics", tool: "get_node_metrics", args: map[string]any{}, wantDen: true},
		{name: "pods-only user denied on list_namespaces", tool: "list_namespaces", args: map[string]any{}, wantDen: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def := toolDefByName(t, tc.tool)
			perms, err := def.permissions(ctx, tctx, tc.args)
			if err != nil {
				t.Fatalf("permissions() returned error: %v", err)
			}
			got := authorizeToolCall(tctx, tctx.cs.Name, perms)
			if tc.wantDen && got == nil {
				t.Fatalf("expected denial, got nil error")
			}
			if !tc.wantDen {
				t.Fatalf("unexpected denial: %v", got)
			}
			if got != nil && !strings.Contains(got.Error(), "forbidden") {
				t.Fatalf("denial must be a Forbidden error, got: %v", got)
			}
		})
	}
}

// TestToolRBACAllowedPaths verifies permissive roles clear the same checks.
func TestToolRBACAllowedPaths(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		tool    string
		args    map[string]any
		user    model.User
		wantDen bool
	}{
		{
			name: "full access passes list_namespaces",
			tool: "list_namespaces", args: map[string]any{},
			user: allAccessUser("admin"),
		},
		{
			name: "pods-only user denied get_node_metrics",
			tool: "get_node_metrics", args: map[string]any{},
			user: podsReadUser(), wantDen: true,
		},
		{
			name: "full access passes list_endpoints with namespace",
			tool: "list_endpoints", args: map[string]any{"namespace": "kube-system"},
			user: allAccessUser("admin"),
		},
		{
			name: "pods-only user denied list_endpoints",
			tool: "list_endpoints", args: map[string]any{"namespace": "kube-system"},
			user: podsReadUser(), wantDen: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def := toolDefByName(t, tc.tool)
			tctx := &toolContext{
				user: tc.user,
				cs:   &cluster.ClientSet{Name: "test"},
				cm:   &cluster.ClusterManager{},
			}
			perms, err := def.permissions(ctx, tctx, tc.args)
			if err != nil {
				t.Fatalf("permissions() returned error: %v", err)
			}
			got := authorizeToolCall(tctx, tctx.cs.Name, perms)
			if tc.wantDen && got == nil {
				t.Fatalf("expected denial, got nil")
			}
			if !tc.wantDen && got != nil {
				t.Fatalf("expected access, got denial: %v", got)
			}
		})
	}
}

// TestToolRegistryAllReadOnly guards the Phase 1 invariant: no write tools.
func TestToolRegistryAllReadOnly(t *testing.T) {
	for _, def := range toolDefs() {
		if def.name == "" || def.description == "" || def.schema == "" {
			t.Fatalf("tool %q missing metadata", def.name)
		}
		var schema map[string]any
		if err := json.Unmarshal([]byte(def.schema), &schema); err != nil {
			t.Fatalf("tool %q has invalid schema: %v", def.name, err)
		}
		if schema["type"] != "object" {
			t.Fatalf("tool %q schema type must be object", def.name)
		}
	}
}

// TestCheckPermissionEvaluatesRBAC checks the check_permission tool
// reports the caller's own access rather than raising a denial.
func TestCheckPermissionEvaluatesRBAC(t *testing.T) {
	def := toolDefByName(t, "check_permission")
	ctx := context.Background()

	allowed := &toolContext{
		user: podsReadUser(),
		cs:   &cluster.ClientSet{Name: "test"},
		cm:   &cluster.ClusterManager{},
	}
	data, err := def.run(ctx, allowed, map[string]any{"verb": "get", "resource": "pods"})
	if err != nil {
		t.Fatalf("run() error: %v", err)
	}
	result := data.(map[string]any)
	if result["allowed"] != true {
		t.Fatalf("expected allowed=true, got %v", result["allowed"])
	}

	denied := &toolContext{
		user: podsReadUser(),
		cs:   &cluster.ClientSet{Name: "test"},
		cm:   &cluster.ClusterManager{},
	}
	data, err = def.run(ctx, denied, map[string]any{"verb": "delete", "resource": "pods"})
	if err != nil {
		t.Fatalf("run() error: %v", err)
	}
	result = data.(map[string]any)
	if result["allowed"] != false {
		t.Fatalf("expected allowed=false, got %v", result["allowed"])
	}
}

func TestNamespaceScope(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{name: "empty means all", args: map[string]any{}, want: ""},
		{name: "_all passes through", args: map[string]any{"namespace": "_all"}, want: ""},
		{name: "named namespace kept", args: map[string]any{"namespace": " default "}, want: "default"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := namespaceScope(tc.args); got != tc.want {
				t.Fatalf("namespaceScope = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter()
	for i := 0; i < 3; i++ {
		if !l.allow(1, 3) {
			t.Fatalf("call %d should be allowed", i+1)
		}
	}
	if l.allow(1, 3) {
		t.Fatal("4th call should be denied at limit 3")
	}
	if !l.allow(2, 3) {
		t.Fatal("another user must have an independent budget")
	}
}
