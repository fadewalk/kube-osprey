package terminal

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/fadewalk/kube-osprey/pkg/kube"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/fadewalk/kube-osprey/pkg/wsutil"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTerminalWebSocketsRejectUnauthorizedUsersBeforeClusterAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	clientSet := &cluster.ClientSet{Name: "prod", K8sClient: &kube.K8sClient{}}
	user := model.User{Username: "alice"}
	router := gin.New()
	router.GET("/terminal/:namespace/:podName", func(c *gin.Context) {
		c.Set("cluster", clientSet)
		c.Set("user", user)
		(&TerminalHandler{}).HandleTerminalWebSocket(c)
	})
	router.GET("/node/:nodeName", func(c *gin.Context) {
		c.Set("cluster", clientSet)
		c.Set("user", user)
		(&NodeTerminalHandler{}).HandleNodeTerminalWebSocket(c)
	})
	router.GET("/kubectl", func(c *gin.Context) {
		c.Set("cluster", clientSet)
		c.Set("user", user)
		(&KubectlTerminalHandler{}).HandleKubectlTerminalWebSocket(c)
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	tests := []struct {
		path        string
		wantMessage string
	}{
		{"/terminal/default/web", "does not have permission to exec pods"},
		{"/node/worker-1", "does not have permission to exec nodes"},
		{"/kubectl", "only available to admin users"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			url := "ws" + strings.TrimPrefix(server.URL, "http") + tt.path
			conn, response, err := websocket.DefaultDialer.Dial(url, nil)
			if response != nil {
				defer func() {
					_ = response.Body.Close()
				}()
			}
			if err != nil {
				if response != nil {
					t.Fatalf("dialing WebSocket: %v, status=%d", err, response.StatusCode)
				}
				t.Fatalf("dialing WebSocket: %v", err)
			}
			defer func() {
				_ = conn.Close()
			}()
			var message wsutil.Message
			if err := conn.ReadJSON(&message); err != nil {
				t.Fatalf("reading rejection: %v", err)
			}
			if message.Type != "error" || !strings.Contains(message.Data, tt.wantMessage) {
				t.Fatalf("rejection message = %#v", message)
			}
		})
	}
}

func TestTerminalAgentsUseExpectedSecurityConfigurationAndCleanupScope(t *testing.T) { //nolint:gocyclo // security configuration test covers both terminal agent types
	originalNamespace := common.AgentPodNamespace
	common.AgentPodNamespace = "kube-osprey-system"
	t.Cleanup(func() {
		common.AgentPodNamespace = originalNamespace
	})

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding core scheme: %v", err)
	}
	if err := rbacv1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding RBAC scheme: %v", err)
	}
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	clientSet := &cluster.ClientSet{
		Name:      "prod",
		K8sClient: &kube.K8sClient{Client: client},
	}
	ctx := context.Background()

	nodeHandler := &NodeTerminalHandler{}
	nodePodName, err := nodeHandler.createNodeAgent(ctx, clientSet, "worker-1", "debug-shell:v1")
	if err != nil {
		t.Fatalf("createNodeAgent() error = %v", err)
	}
	var nodePod corev1.Pod
	if err := client.Get(ctx, types.NamespacedName{Namespace: common.AgentPodNamespace, Name: nodePodName}, &nodePod); err != nil {
		t.Fatalf("loading node agent: %v", err)
	}
	if nodePod.Spec.NodeName != "worker-1" || !nodePod.Spec.HostPID || !nodePod.Spec.HostNetwork || !nodePod.Spec.HostIPC {
		t.Fatalf("node agent host configuration = %#v", nodePod.Spec)
	}
	if len(nodePod.Spec.Containers) != 1 || nodePod.Spec.Containers[0].Image != "debug-shell:v1" {
		t.Fatalf("node agent containers = %#v", nodePod.Spec.Containers)
	}
	container := nodePod.Spec.Containers[0]
	if container.SecurityContext == nil || container.SecurityContext.Privileged == nil || !*container.SecurityContext.Privileged {
		t.Fatalf("node agent is not privileged: %#v", container.SecurityContext)
	}
	if len(container.VolumeMounts) != 1 || container.VolumeMounts[0].MountPath != "/host" {
		t.Fatalf("node agent host mount = %#v", container.VolumeMounts)
	}

	kubectlHandler := &KubectlTerminalHandler{}
	sessionID := "session-a"
	sessionSA := sessionID + "-sa"
	if err := kubectlHandler.ensureSessionAccess(ctx, clientSet, sessionSA, sessionID, model.KubectlPermissionOperator, ""); err != nil {
		t.Fatalf("ensureSessionAccess(operator) error = %v", err)
	}
	var serviceAccount corev1.ServiceAccount
	if err := client.Get(ctx, types.NamespacedName{Namespace: common.AgentPodNamespace, Name: sessionSA}, &serviceAccount); err != nil {
		t.Fatalf("loading session ServiceAccount: %v", err)
	}
	var binding rbacv1.ClusterRoleBinding
	if err := client.Get(ctx, types.NamespacedName{Name: sessionID}, &binding); err != nil {
		t.Fatalf("loading session ClusterRoleBinding: %v", err)
	}
	operatorRoleName := kubectlSessionOperatorRoleName(sessionID)
	if binding.RoleRef.Name != operatorRoleName || len(binding.Subjects) != 1 || binding.Subjects[0].Name != sessionSA {
		t.Fatalf("kubectl ClusterRoleBinding = %#v", binding)
	}
	var operatorRole rbacv1.ClusterRole
	if err := client.Get(ctx, types.NamespacedName{Name: operatorRoleName}, &operatorRole); err != nil {
		t.Fatalf("loading session operator ClusterRole: %v", err)
	}
	for _, rule := range operatorRole.Rules {
		for _, resource := range rule.Resources {
			if resource == "secrets" || resource == "*" {
				t.Fatalf("operator session role must not grant %s: %#v", resource, rule)
			}
		}
	}

	adminSessionID := "session-admin"
	if err := kubectlHandler.ensureSessionAccess(ctx, clientSet, adminSessionID+"-sa", adminSessionID, model.KubectlPermissionAdmin, ""); err != nil {
		t.Fatalf("ensureSessionAccess(admin) error = %v", err)
	}
	var adminBinding rbacv1.ClusterRoleBinding
	if err := client.Get(ctx, types.NamespacedName{Name: adminSessionID}, &adminBinding); err != nil {
		t.Fatalf("loading admin session ClusterRoleBinding: %v", err)
	}
	if adminBinding.RoleRef.Name != "cluster-admin" {
		t.Fatalf("admin session roleRef = %#v", adminBinding.RoleRef)
	}

	existingRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "platform-operator"}}
	if err := client.Create(ctx, existingRole); err != nil {
		t.Fatalf("creating existing ClusterRole: %v", err)
	}
	customSessionID := "session-custom"
	if err := kubectlHandler.ensureSessionAccess(ctx, clientSet, customSessionID+"-sa", customSessionID, model.KubectlPermissionCustom, "platform-operator"); err != nil {
		t.Fatalf("ensureSessionAccess(custom) error = %v", err)
	}
	var customBinding rbacv1.ClusterRoleBinding
	if err := client.Get(ctx, types.NamespacedName{Name: customSessionID}, &customBinding); err != nil {
		t.Fatalf("loading custom session ClusterRoleBinding: %v", err)
	}
	if customBinding.RoleRef.Name != "platform-operator" {
		t.Fatalf("custom session roleRef = %#v", customBinding.RoleRef)
	}
	if err := kubectlHandler.ensureSessionAccess(ctx, clientSet, "session-empty-sa", "session-empty", model.KubectlPermissionCustom, " "); err == nil {
		t.Fatalf("custom mode without a ClusterRole should fail")
	}

	kubectlPodName, err := kubectlHandler.createKubectlAgent(ctx, clientSet, sessionID, sessionSA, "kubectl:v1")
	if err != nil {
		t.Fatalf("createKubectlAgent() error = %v", err)
	}
	var kubectlPod corev1.Pod
	if err := client.Get(ctx, types.NamespacedName{Namespace: common.AgentPodNamespace, Name: kubectlPodName}, &kubectlPod); err != nil {
		t.Fatalf("loading kubectl agent: %v", err)
	}
	if kubectlPod.Spec.ServiceAccountName != sessionSA || kubectlPod.Spec.AutomountServiceAccountToken == nil || !*kubectlPod.Spec.AutomountServiceAccountToken {
		t.Fatalf("kubectl agent service account configuration = %#v", kubectlPod.Spec)
	}
	if len(kubectlPod.Spec.Containers) != 1 || kubectlPod.Spec.Containers[0].Image != "kubectl:v1" {
		t.Fatalf("kubectl agent containers = %#v", kubectlPod.Spec.Containers)
	}

	unrelated := &corev1.Pod{}
	unrelated.Name = "unrelated"
	unrelated.Namespace = common.AgentPodNamespace
	unrelated.Labels = map[string]string{"kite.io/kubectl-session": "session-b"}
	if err := client.Create(ctx, unrelated); err != nil {
		t.Fatalf("creating unrelated pod: %v", err)
	}
	if err := kubectlHandler.cleanupSessionAccess(clientSet, sessionSA, sessionID); err != nil {
		t.Fatalf("cleanupSessionAccess() error = %v", err)
	}
	if err := client.Get(ctx, types.NamespacedName{Namespace: common.AgentPodNamespace, Name: sessionSA}, &corev1.ServiceAccount{}); !apierrors.IsNotFound(err) {
		t.Fatalf("session ServiceAccount still exists or unexpected error: %v", err)
	}
	if err := client.Get(ctx, types.NamespacedName{Name: sessionID}, &rbacv1.ClusterRoleBinding{}); !apierrors.IsNotFound(err) {
		t.Fatalf("session ClusterRoleBinding still exists or unexpected error: %v", err)
	}
	if err := client.Get(ctx, types.NamespacedName{Name: operatorRoleName}, &rbacv1.ClusterRole{}); !apierrors.IsNotFound(err) {
		t.Fatalf("session ClusterRole still exists or unexpected error: %v", err)
	}
	if err := client.Get(ctx, types.NamespacedName{Namespace: common.AgentPodNamespace, Name: unrelated.Name}, &corev1.Pod{}); err != nil {
		t.Fatalf("cleanup removed unrelated pod: %v", err)
	}

	if err := kubectlHandler.cleanupSessionAccess(clientSet, customSessionID+"-sa", customSessionID); err != nil {
		t.Fatalf("cleanupSessionAccess(custom) error = %v", err)
	}
	if err := client.Get(ctx, types.NamespacedName{Name: "platform-operator"}, &rbacv1.ClusterRole{}); err != nil {
		t.Fatalf("cleanup deleted the pre-existing custom ClusterRole: %v", err)
	}
}
