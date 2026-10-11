package system

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	prommodel "github.com/prometheus/common/model"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
	"github.com/zxh326/kite/pkg/utils"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const healthQueryTimeout = 15 * time.Second

// HealthCheck is a stable, code-identified risk item. The frontend owns the
// localized wording so check semantics can evolve without retranslating.
type HealthCheck struct {
	ID       string `json:"id"`
	Category string `json:"category"` // nodes | workloads | storage | platform
	Severity string `json:"severity"` // critical | warning | info
	Count    int    `json:"count"`
	Sample   string `json:"sample,omitempty"`
}

type ControlPlaneComponent struct {
	Name  string `json:"name"`
	Ready int    `json:"ready"`
	Total int    `json:"total"`
}

type NodeUsage struct {
	Node    string  `json:"node"`
	CPUPerc float64 `json:"cpuPercent"` // 0-100, NaN when unknown
	MemPerc float64 `json:"memPercent"`
}

type PodUsage struct {
	Namespace string  `json:"namespace"`
	Pod       string  `json:"pod"`
	CPUCores  float64 `json:"cpuCores"`
	MemBytes  float64 `json:"memBytes"`
}

type HealthReport struct {
	Score             int                     `json:"score"`
	GeneratedAt       time.Time               `json:"generatedAt"`
	PrometheusEnabled bool                    `json:"prometheusEnabled"`
	Checks            []HealthCheck           `json:"checks"`
	ControlPlane      []ControlPlaneComponent `json:"controlPlane"`
	TopNodes          []NodeUsage             `json:"topNodes"`
	TopPods           []PodUsage              `json:"topPods"`
}

func GetHealthReport(c *gin.Context) {
	ctx := c.Request.Context()

	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)
	if !rbac.CanAccessCluster(user, cs.Name) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	checks := []HealthCheck{}
	controlPlane := []ControlPlaneComponent{}
	topNodes := []NodeUsage{}
	topPods := []PodUsage{}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		checks = append(checks, nodeChecks(gctx, cs)...)
		return nil
	})
	g.Go(func() error {
		checks = append(checks, podChecks(gctx, cs)...)
		return nil
	})
	g.Go(func() error {
		checks = append(checks, pvcChecks(gctx, cs)...)
		return nil
	})
	g.Go(func() error {
		controlPlane = controlPlaneChecks(gctx, cs)
		return nil
	})
	g.Go(func() error {
		if cs.PromClient == nil {
			return nil
		}
		qctx, cancel := context.WithTimeout(gctx, healthQueryTimeout)
		defer cancel()
		var promErr error
		topNodes, topPods, promErr = usageTopLists(qctx, cs)
		if promErr != nil {
			checks = append(checks, HealthCheck{
				ID:       "prometheus_unavailable",
				Category: "platform",
				Severity: "warning",
			})
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if cs.PromClient == nil {
		checks = append(checks, HealthCheck{
			ID:       "prometheus_not_configured",
			Category: "platform",
			Severity: "info",
		})
	}

	report := HealthReport{
		Score:             healthScore(checks),
		GeneratedAt:       time.Now(),
		PrometheusEnabled: cs.PromClient != nil,
		Checks:            checks,
		ControlPlane:      controlPlane,
		TopNodes:          topNodes,
		TopPods:           topPods,
	}
	c.JSON(http.StatusOK, report)
}

func appendCheck(checks []HealthCheck, id, category, severity string, count int, sample string) []HealthCheck {
	if count <= 0 {
		return checks
	}
	return append(checks, HealthCheck{
		ID:       id,
		Category: category,
		Severity: severity,
		Count:    count,
		Sample:   sample,
	})
}

func nodeChecks(ctx context.Context, cs *cluster.ClientSet) []HealthCheck {
	var nodes v1.NodeList
	if err := cs.K8sClient.List(ctx, &nodes); err != nil {
		return nil
	}
	var checks []HealthCheck
	for i := range nodes.Items {
		node := &nodes.Items[i]
		ready := false
		var pressures []string
		for _, cond := range node.Status.Conditions {
			switch {
			case cond.Type == v1.NodeReady && cond.Status == v1.ConditionTrue:
				ready = true
			case cond.Status == v1.ConditionTrue &&
				(cond.Type == v1.NodeMemoryPressure || cond.Type == v1.NodeDiskPressure || cond.Type == v1.NodePIDPressure):
				pressures = append(pressures, string(cond.Type))
			}
		}
		if !ready {
			checks = appendCheck(checks, "node_not_ready", "nodes", "critical", 1, node.Name)
		}
		for _, pressure := range pressures {
			switch pressure {
			case string(v1.NodeMemoryPressure):
				checks = appendCheck(checks, "node_memory_pressure", "nodes", "warning", 1, node.Name)
			case string(v1.NodeDiskPressure):
				checks = appendCheck(checks, "node_disk_pressure", "nodes", "warning", 1, node.Name)
			case string(v1.NodePIDPressure):
				checks = appendCheck(checks, "node_pid_pressure", "nodes", "warning", 1, node.Name)
			}
		}
	}
	return checks
}

func podChecks(ctx context.Context, cs *cluster.ClientSet) []HealthCheck {
	var pods v1.PodList
	if err := cs.K8sClient.List(ctx, &pods); err != nil {
		return nil
	}
	var (
		crashloops    int
		crashSample   string
		imagePulls    int
		imageSample   string
		stuckPending  int
		pendingSample string
	)
	now := time.Now()
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
			continue
		}
		if pod.Status.Phase == v1.PodPending && pod.CreationTimestamp.Time.Before(now.Add(-10*time.Minute)) {
			stuckPending++
			if pendingSample == "" {
				pendingSample = pod.Namespace + "/" + pod.Name
			}
			continue
		}
		for _, status := range pod.Status.ContainerStatuses {
			waiting := status.State.Waiting
			if waiting == nil {
				continue
			}
			switch waiting.Reason {
			case "CrashLoopBackOff":
				crashloops++
				if crashSample == "" {
					crashSample = pod.Namespace + "/" + pod.Name
				}
			case "ImagePullBackOff", "ErrImagePull":
				imagePulls++
				if imageSample == "" {
					imageSample = pod.Namespace + "/" + pod.Name
				}
			}
		}
	}
	var checks []HealthCheck
	checks = appendCheck(checks, "pod_crashloop", "workloads", "critical", crashloops, crashSample)
	checks = appendCheck(checks, "pod_image_pull", "workloads", "warning", imagePulls, imageSample)
	checks = appendCheck(checks, "pod_stuck_pending", "workloads", "warning", stuckPending, pendingSample)
	return checks
}

func pvcChecks(ctx context.Context, cs *cluster.ClientSet) []HealthCheck {
	var pvcs v1.PersistentVolumeClaimList
	if err := cs.K8sClient.List(ctx, &pvcs); err != nil {
		return nil
	}
	var (
		pending int
		sample  string
	)
	for i := range pvcs.Items {
		pvc := &pvcs.Items[i]
		if pvc.Status.Phase == v1.ClaimPending || pvc.Status.Phase == v1.ClaimLost {
			pending++
			if sample == "" {
				sample = pvc.Namespace + "/" + pvc.Name
			}
		}
	}
	return appendCheck(nil, "pvc_pending", "storage", "warning", pending, sample)
}

func controlPlaneChecks(ctx context.Context, cs *cluster.ClientSet) []ControlPlaneComponent {
	var pods v1.PodList
	if err := cs.K8sClient.List(ctx, &pods, client.InNamespace("kube-system")); err != nil {
		return nil
	}
	components := map[string]*ControlPlaneComponent{}
	for i := range pods.Items {
		pod := &pods.Items[i]
		component := pod.Labels["component"]
		if component != "kube-apiserver" && component != "kube-scheduler" &&
			component != "kube-controller-manager" && component != "etcd" {
			continue
		}
		entry, ok := components[component]
		if !ok {
			entry = &ControlPlaneComponent{Name: component}
			components[component] = entry
		}
		entry.Total++
		if pod.Status.Phase == v1.PodRunning && utils.IsPodReady(pod) {
			entry.Ready++
		}
	}
	names := make([]string, 0, len(components))
	for name := range components {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]ControlPlaneComponent, 0, len(names))
	for _, name := range names {
		result = append(result, *components[name])
	}
	return result
}

// usageTopLists queries Prometheus for node and pod usage leaders. It never
// fabricates data: when queries fail or return nothing the lists stay empty.
func usageTopLists(ctx context.Context, cs *cluster.ClientSet) ([]NodeUsage, []PodUsage, error) {
	ts := time.Now()

	nodeCPU, _, err := cs.PromClient.Query(ctx, `1 - avg by (node) (rate(node_cpu_seconds_total{mode="idle"}[5m]))`, ts)
	if err != nil {
		return nil, nil, err
	}
	nodeMem, _, err := cs.PromClient.Query(ctx, `sum by (node) (container_memory_working_set_bytes{container!="",container!="POD"}) / sum by (node) (kube_node_status_allocatable{resource="memory"})`, ts)
	if err != nil {
		return nil, nil, err
	}
	nodeMemPerc := vectorPercents(nodeMem)

	nodeUsages := map[string]*NodeUsage{}
	for _, sample := range vectorSamples(nodeCPU) {
		nodeUsages[sample.label] = &NodeUsage{Node: sample.label, CPUPerc: sample.value * 100}
	}
	for node, mem := range nodeMemPerc {
		entry, ok := nodeUsages[node]
		if !ok {
			entry = &NodeUsage{Node: node}
			nodeUsages[node] = entry
		}
		entry.MemPerc = mem
	}
	topNodes := make([]NodeUsage, 0, len(nodeUsages))
	for _, entry := range nodeUsages {
		topNodes = append(topNodes, *entry)
	}
	sort.Slice(topNodes, func(i, j int) bool {
		return max(topNodes[i].CPUPerc, topNodes[i].MemPerc) > max(topNodes[j].CPUPerc, topNodes[j].MemPerc)
	})
	if len(topNodes) > 10 {
		topNodes = topNodes[:10]
	}

	podCPU, _, err := cs.PromClient.Query(ctx, `sum by (namespace, pod) (rate(container_cpu_usage_seconds_total{container!="",container!="POD"}[5m]))`, ts)
	if err != nil {
		return nil, nil, err
	}
	podMem, _, err := cs.PromClient.Query(ctx, `sum by (namespace, pod) (container_memory_working_set_bytes{container!="",container!="POD"})`, ts)
	if err != nil {
		return nil, nil, err
	}

	type podUsageEntry struct {
		cpu float64
		mem float64
	}
	podUsages := map[string]*podUsageEntry{}
	for _, sample := range vectorSamples(podCPU) {
		if podUsages[sample.label] == nil {
			podUsages[sample.label] = &podUsageEntry{}
		}
		podUsages[sample.label].cpu = sample.value
	}
	for _, sample := range vectorSamples(podMem) {
		if podUsages[sample.label] == nil {
			podUsages[sample.label] = &podUsageEntry{}
		}
		podUsages[sample.label].mem = sample.value
	}
	topPods := make([]PodUsage, 0, len(podUsages))
	for label, entry := range podUsages {
		namespace, pod := splitPodLabel(label)
		topPods = append(topPods, PodUsage{
			Namespace: namespace,
			Pod:       pod,
			CPUCores:  entry.cpu,
			MemBytes:  entry.mem,
		})
	}
	sort.Slice(topPods, func(i, j int) bool {
		return topPods[i].CPUCores+topPods[i].MemBytes/1e9 > topPods[j].CPUCores+topPods[j].MemBytes/1e9
	})
	if len(topPods) > 10 {
		topPods = topPods[:10]
	}

	return topNodes, topPods, nil
}

type vectorSample struct {
	label string
	value float64
}

func vectorSamples(value prommodel.Value) []vectorSample {
	vector, ok := value.(prommodel.Vector)
	if !ok {
		return nil
	}
	samples := make([]vectorSample, 0, len(vector))
	for _, s := range vector {
		namespace := string(s.Metric["namespace"])
		pod := string(s.Metric["pod"])
		label := string(s.Metric["node"])
		if namespace != "" && pod != "" {
			label = namespace + "/" + pod
		}
		samples = append(samples, vectorSample{label: label, value: float64(s.Value)})
	}
	return samples
}

func vectorPercents(value prommodel.Value) map[string]float64 {
	result := map[string]float64{}
	for _, sample := range vectorSamples(value) {
		result[sample.label] = sample.value * 100
	}
	return result
}

func splitPodLabel(label string) (namespace, pod string) {
	for i := 0; i < len(label); i++ {
		if label[i] == '/' {
			return label[:i], label[i+1:]
		}
	}
	return "", label
}

// healthScore deducts a fixed penalty per failing check type, floored at 0.
func healthScore(checks []HealthCheck) int {
	score := 100
	for _, check := range checks {
		switch check.Severity {
		case "critical":
			score -= 15
		case "warning":
			score -= 5
		}
	}
	if score < 0 {
		return 0
	}
	return score
}
