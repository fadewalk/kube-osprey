package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/internal"
	"github.com/zxh326/kite/pkg/ai"
	"github.com/zxh326/kite/pkg/changeplan"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/middleware"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/notifypoller"
	"github.com/zxh326/kite/pkg/plugins"
	"github.com/zxh326/kite/pkg/rbac"
	"github.com/zxh326/kite/pkg/scheduler"
	"github.com/zxh326/kite/pkg/syncengine"
	"github.com/zxh326/kite/pkg/templates"
	"k8s.io/klog/v2"
)

var appReady atomic.Bool

func initializeApp(ctx context.Context) (*cluster.ClusterManager, error) {
	appReady.Store(false)
	common.LoadEnvs()
	if err := common.ValidateEncryptionKey(); err != nil {
		return nil, err
	}
	if klog.V(1).Enabled() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := model.InitDB(); err != nil {
		return nil, err
	}
	if _, err := model.GetGeneralSetting(); err != nil {
		return nil, fmt.Errorf("failed to load general setting: %w", err)
	}

	rbac.InitRBAC()
	templates.InitTemplates()
	internal.LoadConfigFromFile(common.ConfigFilePath)
	if common.ConfigFilePath == "" {
		internal.LoadConfigFromEnv()
	}

	cm, err := cluster.NewClusterManager()
	if err != nil {
		return nil, err
	}
	if err := internal.StartConfigWatcher(ctx, common.ConfigFilePath); err != nil {
		klog.Warningf("Failed to watch config file: %v", err)
	}
	scheduler.Start(ctx, cm)
	notifypoller.New(cm).Start(ctx)
	plugins.Start(ctx)
	syncengine.Init(cm)
	ai.SetChangePlanRouter(changeplan.NewService(cm).ChatRouter())
	appReady.Store(true)
	go func() {
		syncengine.ReloadFromSetting()
	}()
	return cm, nil
}

func buildEngine(cm *cluster.ClusterManager) *gin.Engine {
	r := gin.New()
	middleware.ConfigureRawPathRouting(r)
	configureTrustedProxies(r)
	r.Use(middleware.Metrics())
	if !common.DisableGZIP {
		klog.Info("GZIP compression is enabled")
		r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/metrics"})))
	}
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.DevCORS(common.CORSAllowedOrigins))

	base := r.Group(common.Base)
	setupAPIRouter(base, cm)
	setupStatic(r)

	return r
}

func configureTrustedProxies(r *gin.Engine) {
	var trustedProxies []string
	if len(common.TrustedProxies) > 0 {
		trustedProxies = common.TrustedProxies
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		klog.Fatalf("Failed to configure trusted proxies: %v", err)
	}
}
