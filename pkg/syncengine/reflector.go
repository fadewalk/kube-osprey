package syncengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/fadewalk/kube-osprey/pkg/model"
)

const (
	// listPageSize bounds each LIST page during full syncs.
	listPageSize = 500
	// watchTimeoutSeconds restarts watches periodically so the reflector
	// revalidates its resource version.
	watchTimeoutSeconds = int64(300)
	// maxBackoff caps the reconnect delay between failures.
	maxBackoff = 30 * time.Second
	// flushInterval batches watch writes to limit write amplification.
	flushInterval = 250 * time.Millisecond
	// flushThreshold flushes early when the buffer grows large.
	flushThreshold = 100
)

// runReflector keeps one (cluster, resource) type mirrored into the database
// until ctx is cancelled: full list, then watch with periodic relists.
func (m *manager) runReflector(ctx context.Context, generation uint64, cs *cluster.ClientSet, key taskKey) {
	defer func() {
		m.mu.Lock()
		if m.generation == generation {
			delete(m.current, key)
		}
		m.mu.Unlock()
	}()

	meta := common.LookupResource(key.resource)
	if meta == nil {
		trackTask(key.cluster, key.resource, model.SyncTaskError, 0, "unknown resource type")
		return
	}
	gvr := schema.GroupVersionResource{
		Group:    meta.Group,
		Version:  meta.Version,
		Resource: key.resource,
	}
	dyn, err := dynamic.NewForConfig(cs.K8sClient.Configuration)
	if err != nil {
		trackTask(key.cluster, key.resource, model.SyncTaskError, 0, err.Error())
		return
	}

	klog.V(1).Infof("syncengine: starting reflector %s/%s", key.cluster, key.resource)
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		count, latestRV, err := fullList(ctx, dyn, gvr, key)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			klog.Warningf("syncengine: list %s/%s failed: %v", key.cluster, key.resource, err)
			trackTask(key.cluster, key.resource, model.SyncTaskError, count, err.Error())
			if !sleepCtx(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		trackTask(key.cluster, key.resource, model.SyncTaskRunning, count, "")
		backoff = time.Second

		rv := latestRV
		for {
			if ctx.Err() != nil {
				return
			}
			rel := watchOnce(ctx, dyn, gvr, key, &rv)
			switch rel {
			case watchRetry:
				// transient watch error; rv unchanged, re-watch after backoff
				if !sleepCtx(ctx, backoff) {
					return
				}
				backoff = min(backoff*2, maxBackoff)
			case watchRelist:
				// resource version expired or unknown: fall through to list
				klog.V(1).Infof("syncengine: %s/%s relisting (rv %q expired)", key.cluster, key.resource, rv)
				goto relist
			case watchStopped:
				return
			}
		}
	relist:
	}
}

type watchOutcome int

const (
	watchRetry watchOutcome = iota
	watchRelist
	watchStopped
)

// fullList replaces the stored snapshot of (cluster, resource): paginated
// LIST, batched upserts, then soft-deletes rows that vanished.
func fullList(ctx context.Context, dyn dynamic.Interface, gvr schema.GroupVersionResource, key taskKey) (int64, string, error) {
	var all []unstructured.Unstructured
	continueToken := ""
	for {
		opts := metav1.ListOptions{Limit: listPageSize}
		if continueToken != "" {
			opts.Continue = continueToken
		}
		list, err := dyn.Resource(gvr).List(ctx, opts)
		if err != nil {
			return int64(len(all)), "", err
		}
		all = append(all, list.Items...)
		continueToken = list.GetContinue()
		if continueToken == "" {
			break
		}
	}

	rows := make([]*model.SyncObject, 0, len(all))
	latestRV := ""
	for i := range all {
		row, err := syncObjectFromUnstructured(key.cluster, key.resource, &all[i])
		if err != nil {
			continue
		}
		rows = append(rows, row)
		if isRVNewer(all[i].GetResourceVersion(), latestRV) {
			latestRV = all[i].GetResourceVersion()
		}
	}
	if err := upsertObjects(rows); err != nil {
		return int64(len(all)), "", err
	}
	if err := pruneVanished(key.cluster, key.resource, rows); err != nil {
		return int64(len(rows)), latestRV, err
	}
	return int64(len(rows)), latestRV, nil
}

// watchOnce runs a single watch connection from resource version rv. It
// returns watchStopped on shutdown, watchRetry after a transient failure and
// watchRelist when the resource version is no longer served (410 Gone).
func watchOnce(ctx context.Context, dyn dynamic.Interface, gvr schema.GroupVersionResource, key taskKey, rv *string) watchOutcome {
	timeout := watchTimeoutSeconds
	w, err := dyn.Resource(gvr).Watch(ctx, metav1.ListOptions{
		ResourceVersion: *rv,
		TimeoutSeconds:  &timeout,
	})
	if err != nil {
		if isRVExpired(err) {
			return watchRelist
		}
		klog.V(2).Infof("syncengine: watch %s/%s error: %v", key.cluster, key.resource, err)
		return watchRetry
	}
	defer w.Stop()

	buffer := make([]*model.SyncObject, 0, flushThreshold)
	flush := func() {
		if len(buffer) == 0 {
			return
		}
		if err := upsertObjects(buffer); err != nil {
			klog.Warningf("syncengine: flush %s/%s failed: %v", key.cluster, key.resource, err)
		}
		buffer = buffer[:0]
	}
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			flush()
			return watchStopped
		case <-ticker.C:
			flush()
		case ev, ok := <-w.ResultChan():
			if !ok {
				flush()
				// natural watch timeout: re-watch from the latest rv
				trackTask(key.cluster, key.resource, model.SyncTaskRunning, countObjects(key), "")
				return watchRetry
			}
			obj, ok := ev.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}
			switch ev.Type {
			case "ADDED", "MODIFIED":
				if !isRVNewer(obj.GetResourceVersion(), *rv) {
					continue
				}
				row, err := syncObjectFromUnstructured(key.cluster, key.resource, obj)
				if err != nil {
					continue
				}
				*rv = obj.GetResourceVersion()
				buffer = append(buffer, row)
				if len(buffer) >= flushThreshold {
					flush()
				}
			case "DELETED":
				softDeleteObject(key.cluster, string(obj.GetUID()))
				if isRVNewer(obj.GetResourceVersion(), *rv) {
					*rv = obj.GetResourceVersion()
				}
			}
		}
	}
}

// syncObjectFromUnstructured projects an API object into a storage row. The
// raw JSON is stored verbatim minus managedFields noise.
func syncObjectFromUnstructured(clusterName, resource string, obj *unstructured.Unstructured) (*model.SyncObject, error) {
	raw := obj.DeepCopy()
	unstructured.RemoveNestedField(raw.Object, "metadata", "managedFields")
	unstructured.RemoveNestedField(raw.Object, "metadata", "annotations", common.KubectlAnnotation)
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	ownerUIDs := make([]string, 0, len(obj.GetOwnerReferences()))
	for _, ref := range obj.GetOwnerReferences() {
		ownerUIDs = append(ownerUIDs, string(ref.UID))
	}
	ownersJSON, err := json.Marshal(ownerUIDs)
	if err != nil {
		ownersJSON = []byte("[]")
	}

	labels := obj.GetLabels()
	labelsJSON := ""
	if len(labels) > 0 {
		if b, err := json.Marshal(labels); err == nil {
			labelsJSON = string(b)
		}
	}

	creation := obj.GetCreationTimestamp().Time
	if creation.IsZero() {
		creation = time.Now()
	}

	return &model.SyncObject{
		ClusterName:       clusterName,
		UID:               string(obj.GetUID()),
		Group:             obj.GroupVersionKind().Group,
		Version:           obj.GroupVersionKind().Version,
		Kind:              obj.GetKind(),
		Resource:          resource,
		Namespace:         obj.GetNamespace(),
		Name:              obj.GetName(),
		SearchName:        strings.ToLower(obj.GetName()),
		LabelsJSON:        labelsJSON,
		OwnerUIDs:         string(ownersJSON),
		ResourceVersion:   obj.GetResourceVersion(),
		CreationTimestamp: creation,
		ObjectJSON:        string(data),
	}, nil
}

// isRVNewer compares resource versions numerically when possible, falling
// back to string comparison.
func isRVNewer(candidate, current string) bool {
	if current == "" {
		return candidate != ""
	}
	var c, cur uint64
	_, errC := fmt.Sscanf(candidate, "%d", &c)
	_, errCur := fmt.Sscanf(current, "%d", &cur)
	if errC == nil && errCur == nil {
		return c > cur
	}
	return candidate > current
}

func isRVExpired(err error) bool {
	return apierrors.IsGone(err)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
