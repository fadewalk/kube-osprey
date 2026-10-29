package syncengine

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
	"k8s.io/klog/v2"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/fadewalk/kube-osprey/pkg/rbac"
)

// deletedRetention bounds how long soft-deleted rows are kept before the
// physical cleanup removes them.
const deletedRetention = 7 * 24 * time.Hour

// freshness bounds how recently a task must have synced before its data is
// served to consumers; outside the window consumers fall back to direct.
const freshness = 10 * time.Minute

func upsertObjects(rows []*model.SyncObject) error {
	if len(rows) == 0 {
		return nil
	}
	// RV guard: drop rows older than what is already stored.
	uids := make([]string, 0, len(rows))
	for _, row := range rows {
		uids = append(uids, row.UID)
	}
	var stored []model.SyncObject
	if err := model.DB.
		Select("uid", "resource_version", "deleted_at").
		Where("cluster_name = ? AND uid IN ?", rows[0].ClusterName, uids).
		Find(&stored).Error; err != nil {
		return err
	}
	storedRV := make(map[string]string, len(stored))
	for _, s := range stored {
		storedRV[s.UID] = s.ResourceVersion
	}
	fresh := make([]*model.SyncObject, 0, len(rows))
	for _, row := range rows {
		if rv := storedRV[row.UID]; rv != "" && !isRVNewer(row.ResourceVersion, rv) {
			continue
		}
		row.DeletedAt = nil
		fresh = append(fresh, row)
	}
	if len(fresh) == 0 {
		return nil
	}
	return model.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "cluster_name"}, {Name: "uid"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"group", "version", "kind", "resource", "namespace", "name",
			"search_name", "labels_json", "owner_uids", "resource_version",
			"creation_timestamp", "object_json", "deleted_at", "updated_at",
		}),
	}).Create(&fresh).Error
}

func softDeleteObject(clusterName, uid string) {
	now := time.Now()
	if err := model.DB.Model(&model.SyncObject{}).
		Where("cluster_name = ? AND uid = ? AND deleted_at IS NULL", clusterName, uid).
		Updates(map[string]any{"deleted_at": &now}).Error; err != nil {
		klog.Warningf("syncengine: soft delete %s/%s failed: %v", clusterName, uid, err)
	}
}

// pruneVanished soft-deletes stored rows that no longer appear in a fresh
// full list.
func pruneVanished(clusterName, resource string, rows []*model.SyncObject) error {
	var storedUIDs []string
	if err := model.DB.Model(&model.SyncObject{}).
		Where("cluster_name = ? AND resource = ? AND deleted_at IS NULL", clusterName, resource).
		Pluck("uid", &storedUIDs).Error; err != nil {
		return err
	}
	if len(storedUIDs) == 0 {
		return nil
	}
	current := make(map[string]bool, len(rows))
	for _, row := range rows {
		current[row.UID] = true
	}
	now := time.Now()
	vanished := make([]string, 0)
	for _, uid := range storedUIDs {
		if !current[uid] {
			vanished = append(vanished, uid)
		}
	}
	if len(vanished) == 0 {
		return nil
	}
	return model.DB.Model(&model.SyncObject{}).
		Where("cluster_name = ? AND uid IN ? AND deleted_at IS NULL", clusterName, vanished).
		Updates(map[string]any{"deleted_at": &now}).Error
}

func trackTask(clusterName, resource, status string, count int64, errStr string) {
	now := time.Now()
	updates := map[string]any{
		"status":       status,
		"object_count": count,
		"last_error":   errStr,
		"last_sync_at": &now,
	}
	res := model.DB.Model(&model.SyncTask{}).
		Where("cluster_name = ? AND resource = ?", clusterName, resource).
		Updates(updates)
	if res.Error != nil {
		klog.Warningf("syncengine: task update %s/%s failed: %v", clusterName, resource, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		task := model.SyncTask{
			ClusterName: clusterName,
			Resource:    resource,
			Status:      status,
			ObjectCount: count,
			LastError:   errStr,
			LastSyncAt:  &now,
		}
		if err := model.DB.Create(&task).Error; err != nil {
			klog.Warningf("syncengine: task create %s/%s failed: %v", clusterName, resource, err)
		}
	}
}

// CleanupDeleted physically removes soft-deleted rows older than the
// retention window; started with the engine.
func CleanupDeleted(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		cutoff := time.Now().Add(-deletedRetention)
		res := model.DB.
			Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).
			Delete(&model.SyncObject{})
		if res.Error != nil {
			klog.Warningf("syncengine: cleanup failed: %v", res.Error)
		} else if res.RowsAffected > 0 {
			klog.V(1).Infof("syncengine: physically removed %d stale rows", res.RowsAffected)
		}
	}
}

// Active reports whether (cluster, resource) is currently served from the
// database: engine on, task running, and data fresh enough to trust.
func Active(clusterName, resource string) bool {
	var count int64
	cutoff := time.Now().Add(-freshness)
	err := model.DB.Model(&model.SyncTask{}).
		Where("cluster_name = ? AND resource = ? AND status = ? AND last_sync_at > ?",
			clusterName, resource, model.SyncTaskRunning, cutoff).
		Count(&count).Error
	return err == nil && count > 0
}

// Handles reports whether the engine is enabled and the resource is in the
// whitelist, regardless of per-cluster freshness.
func Handles(resource string) bool {
	if defaultManager == nil {
		return false
	}
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	return defaultManager.enabled && defaultManager.res[strings.ToLower(resource)]
}

// ListObjects returns live (non-deleted) objects of one type, optionally
// scoped to a namespace ("" or "_all" = all namespaces), ordered by creation
// time descending to match the direct path's list order.
func ListObjects(clusterName, resource, namespace string) ([]json.RawMessage, error) {
	query := model.DB.Model(&model.SyncObject{}).
		Where("cluster_name = ? AND resource = ? AND deleted_at IS NULL", clusterName, resource)
	if namespace != "" && namespace != "_all" {
		query = query.Where("namespace = ?", namespace)
	}
	var rows []model.SyncObject
	if err := query.Order("creation_timestamp DESC, name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, json.RawMessage(row.ObjectJSON))
	}
	return out, nil
}

// GetObject returns one live object by (cluster, resource, namespace, name).
func GetObject(clusterName, resource, namespace, name string) (json.RawMessage, error) {
	var row model.SyncObject
	query := model.DB.
		Where("cluster_name = ? AND resource = ? AND name = ? AND deleted_at IS NULL", clusterName, resource, name)
	if namespace != "" && namespace != "_all" {
		query = query.Where("namespace = ?", namespace)
	}
	if err := query.First(&row).Error; err != nil {
		return nil, err
	}
	return json.RawMessage(row.ObjectJSON), nil
}

// countObjects returns the number of live rows for a reflector's task record.
func countObjects(key taskKey) int64 {
	var count int64
	model.DB.Model(&model.SyncObject{}).
		Where("cluster_name = ? AND resource = ? AND deleted_at IS NULL", key.cluster, key.resource).
		Count(&count)
	return count
}

// SearchObjectRows matches by lowercase name substring and returns raw rows
// (uid/namespace/name/creation included). Label searches fall back to the
// direct path since labels live inside the JSON blob.
func SearchObjectRows(clusterName, resource, q string, limit int64) ([]model.SyncObject, error) {
	var rows []model.SyncObject
	query := model.DB.
		Where("cluster_name = ? AND resource = ? AND deleted_at IS NULL AND search_name LIKE ?",
			clusterName, resource, "%"+strings.ToLower(q)+"%")
	if err := query.Order("creation_timestamp DESC").Limit(int(limit)).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// TopologyObjects returns the mirrored rows feeding the dependency graph,
// scoped to a namespace (or all).
func TopologyObjects(clusterName, namespace string) ([]model.SyncObject, error) {
	query := model.DB.Model(&model.SyncObject{}).
		Where("cluster_name = ? AND deleted_at IS NULL", clusterName).
		Where("resource IN ?", []string{
			"pods", "deployments", "statefulsets", "daemonsets", "services",
			"replicasets", "jobs", "configmaps", "persistentvolumeclaims",
		})
	if namespace != "" && namespace != "_all" {
		query = query.Where("namespace = ?", namespace)
	}
	var rows []model.SyncObject
	if err := query.Order("creation_timestamp DESC").Limit(5000).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func topologyNode(row model.SyncObject) map[string]any {
	var owners []string
	_ = json.Unmarshal([]byte(row.OwnerUIDs), &owners)
	return map[string]any{
		"uid":       row.UID,
		"kind":      row.Kind,
		"name":      row.Name,
		"namespace": row.Namespace,
		"resource":  row.Resource,
		"ownerUids": owners,
		"createdAt": row.CreationTimestamp,
	}
}

// topologyRef identifies an edge target that may not exist as a synced node
// (e.g. a Secret reference, whose contents the engine deliberately mirrors).
type topologyRef struct {
	kind      string
	namespace string
	name      string
}

func (r topologyRef) placeholderUID() string {
	return "placeholder/" + r.kind + "/" + r.namespace + "/" + r.name
}

// topologyEdge is a non-ownership relation between two graph nodes.
type topologyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // selector | config | storage
}

type topologyPodVolume struct {
	name      string
	configMap string
	secret    string
	pvc       string
}

type topologyPod struct {
	uid       string
	namespace string
	volumes   []topologyPodVolume
	envRefs   []topologyRef
}

type topologyService struct {
	uid       string
	namespace string
	selector  map[string]string
}

// parseTopologyPod extracts the relation-relevant parts of a pod (labels,
// volume sources, env references) from its mirrored JSON.
func parseTopologyPod(row model.SyncObject) (topologyPod, map[string]string) {
	info := topologyPod{uid: row.UID, namespace: row.Namespace}
	var labels map[string]string
	_ = json.Unmarshal([]byte(row.LabelsJSON), &labels)
	var spec struct {
		Spec struct {
			Volumes []struct {
				Name      string `json:"name"`
				ConfigMap *struct {
					Name string `json:"name"`
				} `json:"configMap"`
				Secret *struct {
					SecretName string `json:"secretName"`
				} `json:"secret"`
				PersistentVolumeClaim *struct {
					ClaimName string `json:"claimName"`
				} `json:"persistentVolumeClaim"`
			} `json:"volumes"`
			Containers []struct {
				EnvFrom []struct {
					ConfigMapRef *struct {
						Name string `json:"name"`
					} `json:"configMapRef"`
					SecretRef *struct {
						Name string `json:"name"`
					} `json:"secretRef"`
				} `json:"envFrom"`
				Env []struct {
					ValueFrom *struct {
						ConfigMapKeyRef *struct {
							Name string `json:"name"`
						} `json:"configMapKeyRef"`
						SecretKeyRef *struct {
							Name string `json:"name"`
						} `json:"secretKeyRef"`
					} `json:"valueFrom"`
				} `json:"env"`
			} `json:"containers"`
		} `json:"spec"`
	}
	if row.ObjectJSON == "" || json.Unmarshal([]byte(row.ObjectJSON), &spec) != nil {
		return info, labels
	}
	for _, v := range spec.Spec.Volumes {
		vol := topologyPodVolume{name: v.Name}
		if v.ConfigMap != nil {
			vol.configMap = v.ConfigMap.Name
		}
		if v.Secret != nil {
			vol.secret = v.Secret.SecretName
		}
		if v.PersistentVolumeClaim != nil {
			vol.pvc = v.PersistentVolumeClaim.ClaimName
		}
		info.volumes = append(info.volumes, vol)
	}
	for _, c := range spec.Spec.Containers {
		for _, ef := range c.EnvFrom {
			if ef.ConfigMapRef.Name != "" {
				info.envRefs = append(info.envRefs, topologyRef{kind: "ConfigMap", namespace: row.Namespace, name: ef.ConfigMapRef.Name})
			}
			if ef.SecretRef.Name != "" {
				info.envRefs = append(info.envRefs, topologyRef{kind: "Secret", namespace: row.Namespace, name: ef.SecretRef.Name})
			}
		}
		for _, e := range c.Env {
			if e.ValueFrom == nil {
				continue
			}
			if e.ValueFrom.ConfigMapKeyRef != nil && e.ValueFrom.ConfigMapKeyRef.Name != "" {
				info.envRefs = append(info.envRefs, topologyRef{kind: "ConfigMap", namespace: row.Namespace, name: e.ValueFrom.ConfigMapKeyRef.Name})
			}
			if e.ValueFrom.SecretKeyRef != nil && e.ValueFrom.SecretKeyRef.Name != "" {
				info.envRefs = append(info.envRefs, topologyRef{kind: "Secret", namespace: row.Namespace, name: e.ValueFrom.SecretKeyRef.Name})
			}
		}
	}
	return info, labels
}

// deriveTopologyRelations computes selector (Service→Pod label match),
// config (Pod→ConfigMap/Secret via volumes and env), and storage
// (Pod→PVC via volumes) edges from the mirrored object JSON. Targets that
// are not synced themselves (Secrets) become placeholder nodes carrying no
// data, so the graph stays complete without storing secret contents.
func deriveTopologyRelations(rows []model.SyncObject) (edges []topologyEdge, extraNodes []map[string]any) {
	edges = []topologyEdge{}
	extraNodes = []map[string]any{}

	byRef := map[topologyRef]string{} // existing node lookup: kind+ns+name -> uid
	podLabels := map[string]map[string]string{}
	pods := []topologyPod{}
	services := []topologyService{}
	seenPlaceholder := map[string]bool{}

	addPlaceholder := func(ref topologyRef) string {
		id := ref.placeholderUID()
		if !seenPlaceholder[id] {
			seenPlaceholder[id] = true
			extraNodes = append(extraNodes, map[string]any{
				"uid":         id,
				"kind":        ref.kind,
				"name":        ref.name,
				"namespace":   ref.namespace,
				"resource":    "",
				"ownerUids":   []string{},
				"createdAt":   time.Time{},
				"placeholder": true,
			})
		}
		return id
	}

	for _, row := range rows {
		ref := topologyRef{kind: row.Kind, namespace: row.Namespace, name: row.Name}
		byRef[ref] = row.UID
		switch row.Resource {
		case "pods":
			info, labels := parseTopologyPod(row)
			podLabels[row.UID] = labels
			pods = append(pods, info)
		case "services":
			var spec struct {
				Spec struct {
					Selector map[string]string `json:"selector"`
				} `json:"spec"`
			}
			if row.ObjectJSON != "" && json.Unmarshal([]byte(row.ObjectJSON), &spec) == nil && len(spec.Spec.Selector) > 0 {
				services = append(services, topologyService{
					uid:       row.UID,
					namespace: row.Namespace,
					selector:  spec.Spec.Selector,
				})
			}
		}
	}

	// selector: Service -> Pods whose labels satisfy the selector.
	for _, svc := range services {
		for uid, labels := range podLabels {
			if labels == nil {
				continue
			}
			matched := true
			for k, v := range svc.selector {
				if labels[k] != v {
					matched = false
					break
				}
			}
			if matched {
				edges = append(edges, topologyEdge{From: svc.uid, To: uid, Type: "selector"})
			}
		}
	}

	// config + storage: Pod -> ConfigMap/Secret/PVC.
	seenEdge := map[string]bool{}
	addEdge := func(from, to, typ string) {
		key := from + "|" + to + "|" + typ
		if seenEdge[key] {
			return
		}
		seenEdge[key] = true
		edges = append(edges, topologyEdge{From: from, To: to, Type: typ})
	}
	for _, pod := range pods {
		for _, vol := range pod.volumes {
			if vol.configMap != "" {
				ref := topologyRef{kind: "ConfigMap", namespace: pod.namespace, name: vol.configMap}
				if uid, ok := byRef[ref]; ok {
					addEdge(pod.uid, uid, "config")
				} else {
					addEdge(pod.uid, addPlaceholder(ref), "config")
				}
			}
			if vol.secret != "" {
				ref := topologyRef{kind: "Secret", namespace: pod.namespace, name: vol.secret}
				if uid, ok := byRef[ref]; ok {
					addEdge(pod.uid, uid, "config")
				} else {
					addEdge(pod.uid, addPlaceholder(ref), "config")
				}
			}
			if vol.pvc != "" {
				ref := topologyRef{kind: "PersistentVolumeClaim", namespace: pod.namespace, name: vol.pvc}
				if uid, ok := byRef[ref]; ok {
					addEdge(pod.uid, uid, "storage")
				} else {
					addEdge(pod.uid, addPlaceholder(ref), "storage")
				}
			}
		}
		for _, ref := range pod.envRefs {
			if uid, ok := byRef[ref]; ok {
				addEdge(pod.uid, uid, "config")
			} else {
				addEdge(pod.uid, addPlaceholder(ref), "config")
			}
		}
	}
	return edges, extraNodes
}

// HandleTopology serves the dependency graph (uid + owner chains) for a
// cluster/namespace from the sync tables. It requires cluster-wide pod read
// access, matching what drawing the graph exposes.
func HandleTopology(c *gin.Context) {
	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	u := user.(model.User)
	if !rbac.CanAccess(u, "pods", "get", cs.Name, "_all") {
		c.JSON(http.StatusForbidden, gin.H{"error": "cluster-wide pod read access required"})
		return
	}
	rows, err := TopologyObjects(cs.Name, c.Query("namespace"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	nodes := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		nodes = append(nodes, topologyNode(row))
	}
	edges, extraNodes := deriveTopologyRelations(rows)
	nodes = append(nodes, extraNodes...)
	c.JSON(http.StatusOK, gin.H{
		"objects":   nodes,
		"edges":     edges,
		"available": Active(cs.Name, "pods"),
	})
}
