// Package syncengine mirrors whitelisted Kubernetes resource types into the
// database so lists, global search and topology queries can be served without
// touching the API server. Reflectors are no-store: watch events stream
// straight into sync_objects and nothing is retained in memory, so the
// footprint stays flat no matter how large the cluster is.
//
// The engine is off by default and driven by the general settings; when a
// resource type is not covered (or the engine is off) every consumer falls
// back to the existing direct paths.
package syncengine

import (
	"context"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
)

// DefaultWhitelist is the resource plural list synced when the settings do
// not override it. Secrets are deliberately excluded (an admin may add them
// in the settings; the stored object JSON then contains the secret).
var DefaultWhitelist = []string{
	"pods",
	"deployments",
	"statefulsets",
	"daemonsets",
	"replicasets",
	"jobs",
	"services",
	"ingresses",
	"configmaps",
	"nodes",
	"namespaces",
	"events",
}

type Config struct {
	Enabled   bool
	Resources []string
}

// manager owns the running reflector set and reconciles it against the
// desired configuration.
type manager struct {
	cm *cluster.ClusterManager

	mu      sync.Mutex
	cancel  context.CancelFunc
	current map[taskKey]bool // reflector keys this manager should run
	enabled bool
	res     map[string]bool
	// generation increments on every reload so stale reconcile passes stop.
	generation uint64
}

type taskKey struct {
	cluster  string
	resource string
}

var defaultManager *manager

// Init installs the package-level manager; call once at startup.
func Init(cm *cluster.ClusterManager) {
	defaultManager = &manager{cm: cm}
}

// ReloadFromSetting reads the general setting and reconciles the engine. It
// runs at startup and after every general-setting update.
func ReloadFromSetting() {
	ReloadFromSettingForce(false)
}

// ReloadFromSettingForce behaves like ReloadFromSetting; force=true restarts
// every reflector even when the configuration did not change (full re-sync).
func ReloadFromSettingForce(force bool) {
	if defaultManager == nil {
		return
	}
	cfg := Config{}
	setting, err := model.GetGeneralSetting()
	if err != nil {
		klog.Warningf("syncengine: failed to load general setting, keeping engine off: %v", err)
	} else {
		cfg.Enabled = setting.SyncEngineEnabled
		cfg.Resources = parseWhitelist(setting.SyncEngineResources)
	}
	defaultManager.reload(cfg, force)
}

func parseWhitelist(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultWhitelist
	}
	out := make([]string, 0, 8)
	seen := make(map[string]bool, 8)
	for _, part := range strings.Split(raw, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" && !seen[part] {
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}

func (m *manager) reload(cfg Config, force bool) {
	klog.Infof("syncengine: reload requested (enabled=%v, force=%v, resources=%v)", cfg.Enabled, force, cfg.Resources)
	m.mu.Lock()

	res := make(map[string]bool, len(cfg.Resources))
	for _, r := range cfg.Resources {
		res[r] = true
	}

	if !cfg.Enabled {
		m.stopLocked()
		m.mu.Unlock()
		return
	}
	// Same scope as before and no forced resync: nothing to restart, only
	// bookkeeping changes.
	if !force && m.enabled && m.sameResources(res) {
		m.res = res
		m.mu.Unlock()
		return
	}

	m.stopLocked()
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.enabled = true
	m.res = res
	m.current = make(map[taskKey]bool)
	m.generation++
	gen := m.generation
	m.mu.Unlock()

	go m.reconcileLoop(ctx, gen)
}

func (m *manager) sameResources(res map[string]bool) bool {
	if len(res) != len(m.res) {
		return false
	}
	for k := range res {
		if !m.res[k] {
			return false
		}
	}
	return true
}

func (m *manager) stopLocked() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.enabled = false
	m.res = nil
	m.current = nil
	m.generation++
}

// reconcileLoop periodically reconciles the reflector set so clusters added
// or removed at runtime are picked up without a settings change.
func (m *manager) reconcileLoop(ctx context.Context, generation uint64) {
	m.reconcile(ctx, generation)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			stale := m.generation != generation || !m.enabled
			m.mu.Unlock()
			if stale {
				return
			}
			m.reconcile(ctx, generation)
		}
	}
}

// reconcile starts missing reflectors; removed clusters/resources simply
// lose their marker so their goroutines exit via context cancellation.
func (m *manager) reconcile(ctx context.Context, generation uint64) {
	clusters, _, _ := m.cm.SnapshotState()
	klog.V(1).Infof("syncengine: reconcile gen=%d, %d clusters", generation, len(clusters))
	m.mu.Lock()
	if m.generation != generation || !m.enabled {
		m.mu.Unlock()
		return
	}
	var toStart []taskKey
	csByName := make(map[string]*cluster.ClientSet, len(clusters))
	for name, cs := range clusters {
		csByName[name] = cs
		for resource := range m.res {
			key := taskKey{name, resource}
			if !m.current[key] {
				m.current[key] = true
				toStart = append(toStart, key)
			}
		}
	}
	for key := range m.current {
		if _, ok := clusters[key.cluster]; !ok || !m.res[key.resource] {
			delete(m.current, key)
		}
	}
	var starts []taskKey
	var startCS []*cluster.ClientSet
	for _, key := range toStart {
		if cs := csByName[key.cluster]; cs != nil && cs.K8sClient != nil {
			starts = append(starts, key)
			startCS = append(startCS, cs)
		} else {
			delete(m.current, key)
		}
	}
	m.mu.Unlock()

	for i, key := range starts {
		go m.runReflector(ctx, generation, startCS[i], key)
	}
}
