package resources

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/helmutil"
	"github.com/zxh326/kite/pkg/model"
	"k8s.io/klog/v2"
)

type helmUpgradeWebhookRequest struct {
	Cluster           string                 `json:"cluster" binding:"required"`
	Namespace         string                 `json:"namespace" binding:"required"`
	Release           string                 `json:"release" binding:"required"`
	ChartURL          string                 `json:"chartUrl"`
	RepositoryName    string                 `json:"repositoryName"`
	Source            string                 `json:"source"`
	Values            map[string]interface{} `json:"values"`
	Description       string                 `json:"description"`
	ForceConflicts    bool                   `json:"forceConflicts"`
	RollbackOnFailure bool                   `json:"rollbackOnFailure"`
	Wait              bool                   `json:"wait"`
	DryRun            bool                   `json:"dryRun"`
}

// HelmUpgradeWebhook lets CI systems trigger a helm release upgrade with an
// API key; it reuses the same execution path as the console upgrade action.
func (h *HelmReleaseHandler) HelmUpgradeWebhook(cm *cluster.ClusterManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req helmUpgradeWebhookRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cs, err := cm.GetClientSet(req.Cluster)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("cluster %q not found", req.Cluster)})
			return
		}
		user := c.MustGet("user").(model.User)
		action := helmReleaseActionRequest{
			ChartURL:          req.ChartURL,
			RepositoryName:    req.RepositoryName,
			Source:            req.Source,
			Values:            req.Values,
			Description:       req.Description,
			ForceConflicts:    req.ForceConflicts,
			RollbackOnFailure: req.RollbackOnFailure,
			Wait:              req.Wait,
		}
		result, status, err := h.executeUpgrade(c.Request.Context(), cs, user, req.Namespace, req.Release, action, req.DryRun)
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		if req.DryRun {
			c.JSON(http.StatusOK, helmutil.ToHelmReleaseDryRunDiffResponse(result.current, result.release))
			return
		}
		revision := 0
		if result.release != nil {
			revision = result.release.Version
		}
		klog.V(1).Infof("Webhook upgraded helm release %s/%s (cluster %s) to revision %d by %s", req.Namespace, req.Release, req.Cluster, revision, user.Username)
		c.JSON(http.StatusOK, gin.H{"message": "helm release upgraded", "revision": revision})
	}
}
