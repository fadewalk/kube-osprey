package clusteragent

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// GenerateManifest builds a legacy administrator-compatible manifest.
func GenerateManifest(serverURL, token, publicKey, image string) string {
	return GenerateManifestWithPermission(serverURL, token, publicKey, image, "admin", "")
}

// GenerateManifestWithPermission builds a Cluster Agent manifest with an explicit
// Kubernetes permission profile. Empty permission preserves legacy admin behavior.
func GenerateManifestWithPermission(serverURL, token, publicKey, image, permission, customRole string) string {
	serverURL = strings.TrimSpace(serverURL)
	token = strings.TrimSpace(token)
	publicKey = strings.TrimSpace(publicKey)
	image = strings.TrimSpace(image)
	permission = strings.TrimSpace(permission)
	if permission == "" {
		permission = "admin"
	}
	roleName, roleManifest := agentRole(permission, customRole)

	// JSON-encode string values so YAML remains valid for special characters.
	tokenJSON := fmt.Sprintf("%q", token)
	publicKeyJSON := fmt.Sprintf("%q", publicKey)
	serverJSON := fmt.Sprintf("%q", serverURL)
	imageJSON := fmt.Sprintf("%q", image)
	tokenHashJSON := fmt.Sprintf("%q", tokenHash(token))
	roleNameJSON := roleName
	if len(validation.IsDNS1123Subdomain(roleName)) > 0 {
		roleNameJSON = fmt.Sprintf("%q", roleName)
	}
	bindingManifest := fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kite-cluster-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s
subjects:
  - kind: ServiceAccount
    name: kite-cluster-agent
    namespace: kube-system`, roleNameJSON)
	if roleManifest != "" {
		bindingManifest = roleManifest + "\n---\n" + bindingManifest
	}

	return fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: kite-cluster-agent-token
  namespace: kube-system
type: Opaque
stringData:
  token: %s
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kite-cluster-agent
  namespace: kube-system
---
%s
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kite-cluster-agent
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: kite-cluster-agent
  template:
    metadata:
      annotations:
        kite.kubernetes.io/cluster-agent-token-hash: %s
      labels:
        app.kubernetes.io/name: kite-cluster-agent
    spec:
      serviceAccountName: kite-cluster-agent
      containers:
        - name: cluster-agent
          image: %s
          command:
            - /app/kite
          args:
            - cluster-agent
            - --server=$(KITE_SERVER)
            - --token=$(CLUSTER_AGENT_TOKEN)
            - --public-key=$(CLUSTER_AGENT_PUBLIC_KEY)
          env:
            - name: KITE_SERVER
              value: %s
            - name: CLUSTER_AGENT_TOKEN
              valueFrom:
                secretKeyRef:
                  name: kite-cluster-agent-token
                  key: token
            - name: CLUSTER_AGENT_PUBLIC_KEY
              value: %s
`, tokenJSON, bindingManifest, tokenHashJSON, imageJSON, serverJSON, publicKeyJSON)
}

func agentRole(permission, customRole string) (string, string) {
	switch permission {
	case "readOnly":
		return "kite-cluster-agent-readonly", clusterRoleManifest("kite-cluster-agent-readonly", `- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]`)
	case "operator":
		return "kite-cluster-agent-operator", clusterRoleManifest("kite-cluster-agent-operator", `- apiGroups: [""]
  resources: ["pods", "pods/log", "services", "endpoints", "configmaps", "persistentvolumeclaims"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "networkpolicies"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["policy", "autoscaling"]
  resources: ["poddisruptionbudgets", "horizontalpodautoscalers"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]`)
	case "custom":
		role := strings.TrimSpace(customRole)
		if role != "" {
			return role, ""
		}
		fallthrough
	default:
		return "cluster-admin", ""
	}
}

func clusterRoleManifest(name, rules string) string {
	return fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %q
rules:
%s`, name, rules)
}
