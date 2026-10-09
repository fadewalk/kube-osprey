package mcpserver

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/zxh326/kite/pkg/common"
	corev1 "k8s.io/api/core/v1"
)

func decodeArgs(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}
	return args, nil
}

func requiredString(args map[string]any, key string) (string, error) {
	v, _ := args[key].(string)
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

func optionalString(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return strings.TrimSpace(v)
}

// optionalInt reads an integer argument; JSON numbers arrive as float64.
func optionalInt(args map[string]any, key string, def int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

// namespaceScope normalizes an optional namespace argument to the kube API
// convention: empty or "_all" lists across all namespaces.
func namespaceScope(args map[string]any) string {
	ns := optionalString(args, "namespace")
	if ns == "" || ns == common.AllNamespaces {
		return corev1.NamespaceAll
	}
	return ns
}

// permissionNamespace returns the RBAC scope for an optional namespace:
// a namespaced call without one means all namespaces.
func permissionNamespace(args map[string]any) string {
	ns := optionalString(args, "namespace")
	if ns == "" {
		return common.AllNamespaces
	}
	return ns
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
