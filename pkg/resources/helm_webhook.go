package resources

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/pkg/changeplan"
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

		// Plan mode: route the upgrade through change-plan approval instead
		// of upgrading immediately. Dry runs stay immediate — they only
		// preview the diff.
		if setting, err := model.GetGeneralSetting(); err == nil && setting.WebhookChangePlanMode && !req.DryRun {
			svc := changeplan.NewService(cm)
			description := req.Description
			if description == "" {
				description = fmt.Sprintf("Helm upgrade %s/%s (cluster %s) from webhook", req.Namespace, req.Release, req.Cluster)
			}
			plan, _, err := svc.CreatePlan(user, "webhook", "", description)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create change plan: %v", err)})
				return
			}
			args := map[string]any{
				"namespace":         req.Namespace,
				"release":           req.Release,
				"chartUrl":          req.ChartURL,
				"repositoryName":    req.RepositoryName,
				"source":            req.Source,
				"values":            req.Values,
				"description":       req.Description,
				"forceConflicts":    req.ForceConflicts,
				"rollbackOnFailure": req.RollbackOnFailure,
				"wait":              req.Wait,
			}
			if _, err := svc.AppendStep(c.Request.Context(), user, plan.ID, "helm_upgrade_release", description, args); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to attach upgrade step: %v", err)})
				return
			}
			if err := svc.FinalizePlan(user, plan.ID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to finalize change plan: %v", err)})
				return
			}
			plan.Status = model.ChangePlanPending
			klog.V(1).Infof("Webhook created change plan %d for helm release %s/%s (cluster %s) by %s", plan.ID, req.Namespace, req.Release, req.Cluster, user.Username)
			c.JSON(http.StatusAccepted, gin.H{
				"planId":           plan.ID,
				"approvalRequired": true,
				"status":           plan.Status,
				"message":          "helm upgrade routed through change-plan approval",
			})
			return
		}

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
			c.JSON(http.StatusOK, helmutil.ToHelmReleaseDryRunDiffResponse(result.Current, result.Release))
			return
		}
		revision := 0
		if result.Release != nil {
			revision = result.Release.Version
		}
		klog.V(1).Infof("Webhook upgraded helm release %s/%s (cluster %s) to revision %d by %s", req.Namespace, req.Release, req.Cluster, revision, user.Username)
		c.JSON(http.StatusOK, gin.H{"message": "helm release upgraded", "revision": revision})
	}
}
