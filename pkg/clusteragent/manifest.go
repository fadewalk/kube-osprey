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
  name: kube-osprey-cluster-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s
subjects:
  - kind: ServiceAccount
    name: kube-osprey-cluster-agent
    namespace: kube-system`, roleNameJSON)
	if roleManifest != "" {
		bindingManifest = roleManifest + "\n---\n" + bindingManifest
	}

	return fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: kube-osprey-cluster-agent-token
  namespace: kube-system
type: Opaque
stringData:
  token: %s
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-osprey-cluster-agent
  namespace: kube-system
---
%s
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-osprey-cluster-agent
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: kube-osprey-cluster-agent
  template:
    metadata:
      annotations:
        kube-osprey.io/cluster-agent-token-hash: %s
      labels:
        app.kubernetes.io/name: kube-osprey-cluster-agent
    spec:
      serviceAccountName: kube-osprey-cluster-agent
      containers:
        - name: cluster-agent
          image: %s
          command:
            - /app/kube-osprey
          args:
            - cluster-agent
            - --server=$(KUBE_OSPREY_SERVER)
            - --token=$(CLUSTER_AGENT_TOKEN)
            - --public-key=$(CLUSTER_AGENT_PUBLIC_KEY)
          env:
            - name: KUBE_OSPREY_SERVER
              value: %s
            - name: CLUSTER_AGENT_TOKEN
              valueFrom:
                secretKeyRef:
                  name: kube-osprey-cluster-agent-token
                  key: token
            - name: CLUSTER_AGENT_PUBLIC_KEY
              value: %s
`, tokenJSON, bindingManifest, tokenHashJSON, imageJSON, serverJSON, publicKeyJSON)
}

func agentRole(permission, customRole string) (string, string) {
	switch permission {
	case "readOnly":
		return "kube-osprey-cluster-agent-readonly", clusterRoleManifest("kube-osprey-cluster-agent-readonly", `- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]`)
	case "operator":
		return "kube-osprey-cluster-agent-operator", clusterRoleManifest("kube-osprey-cluster-agent-operator", `- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["pods/log", "pods/exec", "pods/attach", "pods/portforward"]
  verbs: ["get", "create"]
- apiGroups: [""]
  resources: ["services", "endpoints", "configmaps", "persistentvolumeclaims"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["events", "namespaces", "nodes", "persistentvolumes"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
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
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]
- apiGroups: ["apiextensions.k8s.io"]
  resources: ["customresourcedefinitions"]
  verbs: ["get", "list", "watch"]`)
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
