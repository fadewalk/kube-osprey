package system

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
	v1 "k8s.io/api/core/v1"
)

type MultiClusterOverviewHandler struct {
	cm *cluster.ClusterManager
}

func NewMultiClusterOverviewHandler(cm *cluster.ClusterManager) *MultiClusterOverviewHandler {
	return &MultiClusterOverviewHandler{cm: cm}
}

type clusterOverviewSummary struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
	Version       string `json:"version,omitempty"`
	TotalNodes    int    `json:"totalNodes"`
	ReadyNodes    int    `json:"readyNodes"`
	TotalPods     int    `json:"totalPods"`
	RunningPods   int    `json:"runningPods"`
	TotalNS       int    `json:"totalNamespaces"`
	TotalServices int    `json:"totalServices"`
}

// GetMultiClusterOverview aggregates a light per-cluster summary across every
// enabled cluster the user may access; unreachable clusters report status
// "error" instead of failing the whole response.
func (h *MultiClusterOverviewHandler) GetMultiClusterOverview(c *gin.Context) {
	user := c.MustGet("user").(model.User)
	clients, _, _ := h.cm.SnapshotState()

	names := make([]string, 0, len(clients))
	for name := range clients {
		if rbac.CanAccessCluster(user, name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	summaries := make([]clusterOverviewSummary, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			summaries[i] = summarizeCluster(c.Request.Context(), clients[name], name)
		}(i, name)
	}
	wg.Wait()

	c.JSON(http.StatusOK, summaries)
}

func summarizeCluster(ctx context.Context, cs *cluster.ClientSet, name string) clusterOverviewSummary {
	summary := clusterOverviewSummary{Name: name, Status: "ok", Version: cs.Version}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	// One failing list marks the cluster degraded but keeps partial counts.
	fail := func(err error) {
		if err != nil {
			mu.Lock()
			summary.Status = "error"
			if summary.Error == "" {
				summary.Error = err.Error()
			}
			mu.Unlock()
		}
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		var nodes v1.NodeList
		if err := cs.K8sClient.List(ctx, &nodes); err != nil {
			fail(err)
			return
		}
		mu.Lock()
		summary.TotalNodes = len(nodes.Items)
		mu.Unlock()
		for i := range nodes.Items {
			for _, cond := range nodes.Items[i].Status.Conditions {
				if cond.Type == v1.NodeReady && cond.Status == v1.ConditionTrue {
					mu.Lock()
					summary.ReadyNodes++
					mu.Unlock()
					break
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		var pods v1.PodList
		if err := cs.K8sClient.List(ctx, &pods); err != nil {
			fail(err)
			return
		}
		running := 0
		for i := range pods.Items {
			phase := pods.Items[i].Status.Phase
			if phase != v1.PodSucceeded && phase != v1.PodFailed {
				running++
			}
		}
		mu.Lock()
		summary.TotalPods = len(pods.Items)
		summary.RunningPods = running
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		var namespaces v1.NamespaceList
		var services v1.ServiceList
		var wg2 sync.WaitGroup
		wg2.Add(2)
		go func() {
			defer wg2.Done()
			if err := cs.K8sClient.List(ctx, &namespaces); err != nil {
				fail(err)
				return
			}
			mu.Lock()
			summary.TotalNS = len(namespaces.Items)
			mu.Unlock()
		}()
		go func() {
			defer wg2.Done()
			if err := cs.K8sClient.List(ctx, &services); err != nil {
				fail(err)
				return
			}
			mu.Lock()
			summary.TotalServices = len(services.Items)
			mu.Unlock()
		}()
		wg2.Wait()
	}()
	wg.Wait()

	return summary
}
