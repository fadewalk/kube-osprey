package resources

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/fadewalk/kube-osprey/pkg/rbac"
)

// workloadKinds maps the URL kind to its apps/v1 Kind. Only scalable
// template-based workloads support image updates and rollout restarts.
var workloadKinds = map[string]string{
	"deployments":  "Deployment",
	"statefulsets": "StatefulSet",
	"daemonsets":   "DaemonSet",
}

type workloadOpRequest struct {
	Kind      string `json:"kind" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Container string `json:"container"`
	Image     string `json:"image"`
}

func workloadOpRBAC(c *gin.Context, cs *cluster.ClientSet, kind, namespace string) (model.User, bool) {
	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return model.User{}, false
	}
	u := user.(model.User)
	if !rbac.CanAccess(u, kind, string(common.VerbUpdate), cs.Name, namespace) {
		c.JSON(http.StatusForbidden, gin.H{"error": rbac.NoAccess(u.Key(), "update", kind, namespace, cs.Name)})
		return model.User{}, false
	}
	return u, true
}

func getWorkloadUnstructured(c *gin.Context, cs *cluster.ClientSet, req *workloadOpRequest) (*unstructured.Unstructured, error) {
	kind, ok := workloadKinds[req.Kind]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind must be one of deployments, statefulsets, daemonsets"})
		return nil, fmt.Errorf("invalid kind")
	}
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind(kind)
	if err := cs.K8sClient.Get(c.Request.Context(), k8stypes.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// UpdateWorkloadImage patches one (or all) container images of a workload —
// the semantic equivalent of Kuboard's 调整镜像版本 frequent operation.
func UpdateWorkloadImage(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	var req workloadOpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Image = strings.TrimSpace(req.Image)
	if req.Image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image is required"})
		return
	}
	user, ok := workloadOpRBAC(c, cs, req.Kind, req.Namespace)
	if !ok {
		return
	}

	obj, err := getWorkloadUnstructured(c, cs, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	if err != nil || !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read workload containers"})
		return
	}
	updated := 0
	for i, raw := range containers {
		container, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := container["name"].(string)
		if req.Container != "" && name != req.Container {
			continue
		}
		container["image"] = req.Image
		containers[i] = container
		updated++
	}
	if updated == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("container %q not found", req.Container)})
		return
	}
	if err := unstructured.SetNestedSlice(obj.Object, containers, "spec", "template", "spec", "containers"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := cs.K8sClient.Update(c.Request.Context(), obj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	klog.V(1).Infof("Image of %s %s/%s updated to %s by %s", req.Kind, req.Namespace, req.Name, req.Image, user.Username)
	writeWorkloadOpHistory(cs, &user, req.Kind, req.Name, req.Namespace, "update", true, "")
	c.JSON(http.StatusOK, gin.H{"status": "updated", "containers": updated})
}

// RestartWorkload performs a rollout restart via the restartedAt annotation
// for deployments, statefulsets and daemonsets.
func RestartWorkload(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	var req workloadOpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, ok := workloadOpRBAC(c, cs, req.Kind, req.Namespace)
	if !ok {
		return
	}

	obj, err := getWorkloadUnstructured(c, cs, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	annotations, _, err := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
	if err != nil {
		annotations = map[string]string{}
	}
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[common.KubectlAnnotation] = ""
	annotations["kube-osprey.io/restartedAt"] = time.Now().Format(time.RFC3339)
	if err := unstructured.SetNestedStringMap(obj.Object, annotations, "spec", "template", "metadata", "annotations"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := cs.K8sClient.Update(c.Request.Context(), obj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	klog.V(1).Infof("%s %s/%s restart requested by %s", req.Kind, req.Namespace, req.Name, user.Username)
	writeWorkloadOpHistory(cs, &user, req.Kind, req.Name, req.Namespace, "restart", true, "")
	c.JSON(http.StatusOK, gin.H{"status": "restart initiated"})
}

// writeWorkloadOpHistory records a workload-op into the resource history so
// UI-driven restarts and image updates are auditable like every other change.
func writeWorkloadOpHistory(cs *cluster.ClientSet, user *model.User, kind, name, namespace, opType string, success bool, errMsg string) {
	history := model.ResourceHistory{
		ClusterName:     cs.Name,
		ResourceType:    kind,
		ResourceName:    name,
		Namespace:       namespace,
		OperationType:   opType,
		OperationSource: "manual",
		Success:         success,
		ErrorMessage:    errMsg,
		OperatorID:      user.ID,
	}
	if err := model.DB.Create(&history).Error; err != nil {
		klog.Warningf("Failed to record workload op history: %v", err)
	}
}
