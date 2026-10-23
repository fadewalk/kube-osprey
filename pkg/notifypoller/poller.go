// Package notifypoller watches Alertmanager alerts and kubernetes warning
// events across clusters and pushes newly observed ones to channels via
// pkg/notify. Seen fingerprints are kept in memory with a TTL: the first
// poll after startup only primes the set, so a restart does not replay
// everything currently firing.
package notifypoller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zxh326/kite/pkg/alertmanager"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/notify"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	pollInterval = time.Minute
	seenTTL      = 24 * time.Hour
	// Only events touched in the recent window are considered; the seen
	// set handles dedup, this just bounds the working set.
	eventWindow = 5 * time.Minute
	maxEvents   = 200
	maxText     = 2000
)

type Poller struct {
	cm *cluster.ClusterManager

	mu   sync.Mutex
	seen map[string]time.Time
}

func New(cm *cluster.ClusterManager) *Poller {
	return &Poller{cm: cm, seen: map[string]time.Time{}}
}

// Start runs the poll loop until ctx is cancelled.
func (p *Poller) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.pollOnce(ctx)
			}
		}
	}()
}

func (p *Poller) pollOnce(ctx context.Context) {
	wantAlerts := notify.HasSubscriber(model.NotifyEventAlertFiring)
	wantEvents := notify.HasSubscriber(model.NotifyEventEventWarning)
	if !wantAlerts && !wantEvents {
		return
	}

	clusters, err := model.ListClusters()
	if err != nil {
		klog.Errorf("notifypoller: failed to list clusters: %v", err)
		return
	}
	for _, c := range clusters {
		cs, err := p.cm.GetClientSet(c.Name)
		if err != nil {
			continue
		}
		if wantAlerts {
			p.pollAlerts(ctx, c.Name, cs)
		}
		if wantEvents {
			p.pollEvents(ctx, c.Name, cs)
		}
	}
	p.forgetStale()
}

func (p *Poller) pollAlerts(ctx context.Context, clusterName string, cs *cluster.ClientSet) {
	if cs.AlertClient == nil {
		return
	}
	alerts, err := cs.AlertClient.ListAlerts(ctx)
	if err != nil {
		klog.V(2).Infof("notifypoller: failed to list alerts for cluster %s: %v", clusterName, err)
		return
	}

	primed := p.primed("alerts:" + clusterName)
	for _, alert := range alerts {
		// Silenced alerts are muted on purpose; only fire on live ones.
		if alert.Status.State != "active" || len(alert.Status.SilencedBy) > 0 {
			continue
		}
		fp := alertFingerprint(clusterName, alert)
		if !p.markSeen(fp) {
			continue
		}
		if !primed {
			continue
		}
		notify.Dispatch(model.NotifyEventAlertFiring,
			fmt.Sprintf("[%s] Alert %s is firing", clusterName, alert.Labels["alertname"]),
			alertText(clusterName, alert))
	}
}

func (p *Poller) pollEvents(ctx context.Context, clusterName string, cs *cluster.ClientSet) {
	events, err := cs.K8sClient.ClientSet.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
		Limit:         maxEvents,
	})
	if err != nil {
		klog.V(2).Infof("notifypoller: failed to list events for cluster %s: %v", clusterName, err)
		return
	}

	primed := p.primed("events:" + clusterName)
	cutoff := time.Now().Add(-eventWindow)
	for i := range events.Items {
		event := &events.Items[i]
		last := event.LastTimestamp.Time
		if last.IsZero() {
			last = event.FirstTimestamp.Time
		}
		if last.Before(cutoff) {
			continue
		}
		fp := eventFingerprint(clusterName, event)
		if !p.markSeen(fp) {
			continue
		}
		if !primed {
			continue
		}
		target := fmt.Sprintf("%s/%s", event.Namespace, event.InvolvedObject.Name)
		title := fmt.Sprintf("[%s] Warning %s on %s", clusterName, event.Reason, target)
		text := event.Message
		if event.Count > 1 {
			text += fmt.Sprintf("\n> occurred %d times, most recent %s", event.Count, last.Format(time.RFC3339))
		}
		notify.Dispatch(model.NotifyEventEventWarning, title, clip(text))
	}
}

// primed reports whether the source already has a baseline: the first call
// for a source returns false so its observations are recorded silently.
func (p *Poller) primed(source string) bool {
	return p.markSeen("primed:" + source)
}

// markSeen records fingerprint and reports whether it was already known.
func (p *Poller) markSeen(fingerprint string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, known := p.seen[fingerprint]
	p.seen[fingerprint] = time.Now()
	return known
}

func (p *Poller) forgetStale() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for fp, at := range p.seen {
		if time.Since(at) > seenTTL {
			delete(p.seen, fp)
		}
	}
}

func alertFingerprint(clusterName string, alert alertmanager.Alert) string {
	if alert.Fingerprint != "" {
		return clusterName + "/alert/" + alert.Fingerprint
	}
	// Stable fallback when Alertmanager does not supply one.
	keys := make([]string, 0, len(alert.Labels))
	for k := range alert.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(clusterName)
	for _, k := range keys {
		b.WriteString("|" + k + "=" + alert.Labels[k])
	}
	sum := sha256.Sum256([]byte(b.String()))
	return clusterName + "/alert/" + hex.EncodeToString(sum[:8])
}

func alertText(clusterName string, alert alertmanager.Alert) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Cluster: %s\n", clusterName)
	fmt.Fprintf(&b, "Alert: %s\n", alert.Labels["alertname"])
	if ns := alert.Labels["namespace"]; ns != "" {
		fmt.Fprintf(&b, "Namespace: %s\n", ns)
	}
	if sev := alert.Labels["severity"]; sev != "" {
		fmt.Fprintf(&b, "Severity: %s\n", sev)
	}
	fmt.Fprintf(&b, "Started: %s\n", alert.StartsAt.Format(time.RFC3339))
	if desc := alert.Annotations["description"]; desc != "" {
		fmt.Fprintf(&b, "\n%s", desc)
	} else if summary := alert.Annotations["summary"]; summary != "" {
		fmt.Fprintf(&b, "\n%s", summary)
	}
	return clip(b.String())
}

func eventFingerprint(clusterName string, event *corev1.Event) string {
	// UID dedups per event lifecycle: aggregated events share one UID, so
	// a repeating warning notifies once even as its count grows.
	if event.UID != "" {
		return clusterName + "/event/" + string(event.UID)
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s",
		clusterName, event.Namespace, event.InvolvedObject.Name, event.Message)))
	return clusterName + "/event/" + hex.EncodeToString(sum[:8])
}

func clip(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxText {
		return s[:maxText]
	}
	return s
}
