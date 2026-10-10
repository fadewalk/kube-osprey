package syncengine

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/zxh326/kite/pkg/model"
)

// HandleStatus serves the sync engine's self-observation: one row per
// (cluster, resource) task with status, object count, freshness and errors.
func HandleStatus(c *gin.Context) {
	var tasks []model.SyncTask
	if err := model.DB.Order("cluster_name ASC, resource ASC").Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	enabled := defaultManager != nil
	if enabled {
		defaultManager.mu.Lock()
		enabled = defaultManager.enabled
		defaultManager.mu.Unlock()
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled": enabled,
		"tasks":   tasks,
	})
}

// HandleRestartFullSync forces every reflector to restart: cancel + fresh
// LIST + watch, without touching the configuration.
func HandleRestartFullSync(c *gin.Context) {
	if defaultManager == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "sync engine is not initialized"})
		return
	}
	ReloadFromSettingForce(true)
	klog.Info("syncengine: full resync requested via API")
	c.JSON(http.StatusOK, gin.H{"status": "restarting"})
}
