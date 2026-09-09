package terminal

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/kube"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
	"github.com/zxh326/kite/pkg/utils"
	"github.com/zxh326/kite/pkg/wsutil"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/zxh326/kite/pkg/common"
)

const kubectlSessionRole = "kite-kubectl-session"

func kubectlSessionServiceAccountName(instanceID string) string {
	name := instanceID + "-sa"
	if len(name) > 63 {
		name = strings.TrimRight(name[:63], "-.")
	}
	return name
}

type KubectlTerminalHandler struct {
}

func NewKubectlTerminalHandler() *KubectlTerminalHandler {
	return &KubectlTerminalHandler{}
}

func (h *KubectlTerminalHandler) HandleKubectlTerminalWebSocket(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)

	wsutil.Serve(c.Writer, c.Request, func(ws *wsutil.Session) {
		// Only admin users can use the kubectl terminal
		if !rbac.UserHasRole(user, model.DefaultAdminRole.Name) {
			ws.SendErrorMessage("kubectl terminal is only available to admin users")
			return
		}

		setting, err := model.GetGeneralSetting()
		if err != nil {
			ws.SendErrorMessage(fmt.Sprintf("Failed to load settings: %v", err))
			return
		}
		if !setting.KubectlEnabled {
			ws.SendErrorMessage("kubectl terminal is disabled")
			return
		}
		kubectlImage := strings.TrimSpace(setting.KubectlImage)
		if kubectlImage == "" {
			kubectlImage = common.KubectlTerminalImage
		}

		ctx := ws.Context

		instanceID := utils.GenerateKubectlAgentName(user.Key())
		sessionSA := kubectlSessionServiceAccountName(instanceID)
		if err := h.ensureSessionAccess(ctx, cs, sessionSA, instanceID); err != nil {
			klog.Errorf("Failed to setup kubectl session access: %v", err)
			ws.SendErrorMessage(fmt.Sprintf("Failed to setup kubectl terminal: %v", err))
			return
		}

		podName, err := h.createKubectlAgent(ctx, cs, instanceID, sessionSA, kubectlImage)
		if err != nil {
			klog.Errorf("Failed to create kubectl agent pod: %v", err)
			ws.SendErrorMessage(fmt.Sprintf("Failed to create kubectl agent pod: %v", err))
			_ = h.cleanupSessionAccess(cs, sessionSA, instanceID)
			return

		}

		defer func() {
			klog.Infof("Cleaning up kubectl pod %s", instanceID)
			if err := h.cleanupSessionAccess(cs, sessionSA, instanceID); err != nil {
				klog.Errorf("Failed to cleanup kubectl session access: %v", err)
			}
		}()

		if err := waitForAgentPodReady(ctx, cs, ws, podName, "kubectl agent ready!"); err != nil {
			klog.Errorf("Failed to wait for kubectl agent pod ready: %v", err)
			ws.SendErrorMessage(fmt.Sprintf("Failed to wait for kubectl agent pod ready: %v", err))
			return
		}

		session := kube.NewTerminalSession(cs.K8sClient, ws.Conn, common.AgentPodNamespace, podName, common.KubectlTerminalPodName)
		if err := session.Start(ctx, "attach"); err != nil {
			klog.Errorf("Kubectl terminal session error: %v", err)
		}
	})
}

func (h *KubectlTerminalHandler) ensureSessionAccess(ctx context.Context, cs *cluster.ClientSet, serviceAccountName, sessionID string) error {
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "kite",
		"kite.io/component":            "kubectl-terminal",
		"kite.io/kubectl-session":      sessionID,
	}
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: common.AgentPodNamespace, Labels: labels}}
	if err := cs.K8sClient.Create(ctx, sa); err != nil {
		return fmt.Errorf("failed to create session ServiceAccount: %w", err)
	}
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: sessionID, Labels: labels},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: serviceAccountName, Namespace: common.AgentPodNamespace}},
		RoleRef:    rbacv1.RoleRef{Kind: "ClusterRole", Name: "cluster-admin", APIGroup: "rbac.authorization.k8s.io"},
	}
	if err := cs.K8sClient.Create(ctx, crb); err != nil {
		_ = cs.K8sClient.Delete(ctx, sa)
		return fmt.Errorf("failed to create session ClusterRoleBinding: %w", err)
	}
	return nil
}

func (h *KubectlTerminalHandler) cleanupSessionAccess(cs *cluster.ClientSet, serviceAccountName, sessionID string) error {
	ctx := context.TODO()
	if err := cs.K8sClient.Delete(ctx, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: sessionID}}); client.IgnoreNotFound(err) != nil {
		return err
	}
	if err := cs.K8sClient.Delete(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: serviceAccountName, Namespace: common.AgentPodNamespace}}); client.IgnoreNotFound(err) != nil {
		return err
	}
	return h.cleanupPod(cs, sessionID)
}

func (h *KubectlTerminalHandler) createKubectlAgent(ctx context.Context, cs *cluster.ClientSet, instanceID, serviceAccountName, image string) (string, error) {
	podName := instanceID

	gracePeriod := int64(0)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: common.AgentPodNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kite",
				"kite.io/component":            "kubectl-terminal",
				"kite.io/kubectl-session":      instanceID,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyNever,
			ServiceAccountName:            serviceAccountName,
			AutomountServiceAccountToken:  &[]bool{true}[0],
			Hostname:                      "kubectl",
			TerminationGracePeriodSeconds: &gracePeriod,
			Containers: []corev1.Container{
				{
					Name:            common.KubectlTerminalPodName,
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Stdin:           true,
					StdinOnce:       true,
					TTY:             true,
					Command:         []string{"bash", "-c", `exec bash`},
				},
			},
		},
	}

	if err := cs.K8sClient.Create(ctx, pod); err != nil {
		return "", fmt.Errorf("failed to create kubectl agent pod: %w", err)
	}

	return podName, nil
}

// cleanupPod deletes only the per-session pod (the admin SA/CRB are permanent).
func (h *KubectlTerminalHandler) cleanupPod(cs *cluster.ClientSet, instanceID string) error {
	ctx := context.TODO()
	opts := []client.DeleteAllOfOption{
		client.InNamespace(common.AgentPodNamespace),
		client.MatchingLabels{"kite.io/kubectl-session": instanceID},
		client.PropagationPolicy(metav1.DeletePropagationBackground),
	}
	return cs.K8sClient.DeleteAllOf(ctx, &corev1.Pod{}, opts...)
}
