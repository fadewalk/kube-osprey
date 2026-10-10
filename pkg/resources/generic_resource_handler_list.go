package resources

import (
	"encoding/json"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/common"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
	"github.com/zxh326/kite/pkg/syncengine"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// listFromSync serves a list from the sync engine tables, mirroring the
// direct path's RBAC filtering and ordering. Any error returns the caller to
// the direct path.
func (h *GenericResourceHandler[T, V]) listFromSync(c *gin.Context, objectList V, namespace string, allNamespaces bool) (V, error) {
	var zero V
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)

	raws, err := syncengine.ListObjects(cs.Name, h.name, namespace)
	if err != nil {
		return zero, err
	}
	items := make([]runtime.Object, 0, len(raws))
	for _, raw := range raws {
		item := reflect.New(h.objectType).Interface().(T)
		if err := json.Unmarshal(raw, item); err != nil {
			return zero, err
		}
		obj, err := meta.Accessor(item)
		if err != nil {
			return zero, err
		}
		if h.Name() == string(common.Namespaces) && !rbac.CanAccessNamespace(user, cs.Name, obj.GetName()) {
			continue
		}
		if allNamespaces && obj.GetNamespace() != "" &&
			!rbac.CanAccess(user, h.Name(), string(common.VerbGet), cs.Name, obj.GetNamespace()) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		o1, _ := meta.Accessor(items[i])
		o2, _ := meta.Accessor(items[j])
		if o1 == nil || o2 == nil {
			return false
		}
		t1 := o1.GetCreationTimestamp()
		t2 := o2.GetCreationTimestamp()
		if t1.Equal(&t2) {
			return o1.GetName() < o2.GetName()
		}
		return t1.After(t2.Time)
	})
	if err := meta.SetList(objectList, items); err != nil {
		return zero, err
	}
	return objectList, nil
}

func (h *GenericResourceHandler[T, V]) list(c *gin.Context) (V, error) {
	var zero V
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	objectList := reflect.New(h.listType).Interface().(V)

	ctx := c.Request.Context()

	namespace := c.Param("namespace")
	allNamespaces := namespace == "" || namespace == common.AllNamespaces

	// Sync engine fast path: whitelisted types with fresh data are served
	// from the database; any failure falls back to the direct path.
	if syncengine.Handles(h.name) && syncengine.Active(cs.Name, h.name) {
		if served, err := h.listFromSync(c, objectList, namespace, allNamespaces); err == nil {
			return served, nil
		} else {
			klog.V(2).Infof("sync path for %s failed, falling back to direct: %v", h.name, err)
		}
	}

	var listOpts []client.ListOption
	if !h.isClusterScoped {
		if !allNamespaces {
			listOpts = append(listOpts, client.InNamespace(namespace))
		}
	}
	if c.Query("limit") != "" {
		limit, err := strconv.ParseInt(c.Query("limit"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
			return zero, err
		}
		listOpts = append(listOpts, client.Limit(limit))
	}

	if c.Query("continue") != "" {
		continueToken := c.Query("continue")
		listOpts = append(listOpts, client.Continue(continueToken))
	}

	if c.Query("labelSelector") != "" {
		labelSelector := c.Query("labelSelector")
		selector, err := metav1.ParseToLabelSelector(labelSelector)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid labelSelector parameter: " + err.Error()})
			return zero, err
		}
		labelSelectorOption, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to convert labelSelector: " + err.Error()})
			return zero, err
		}
		listOpts = append(listOpts, client.MatchingLabelsSelector{Selector: labelSelectorOption})
	}

	if c.Query("fieldSelector") != "" {
		fieldSelector := c.Query("fieldSelector")
		fieldSelectorOption, err := fields.ParseSelector(fieldSelector)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fieldSelector parameter: " + err.Error()})
			return zero, err
		}
		listOpts = append(listOpts, client.MatchingFieldsSelector{Selector: fieldSelectorOption})
	}

	if err := cs.K8sClient.List(ctx, objectList, listOpts...); err != nil {
		if h.Name() == string(common.EndpointSlices) && meta.IsNoMatchError(err) {
			_ = meta.SetList(objectList, []runtime.Object{})
			return objectList, nil
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return zero, err
	}

	items, err := meta.ExtractList(objectList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to extract items from list"})
		return zero, err
	}
	sort.Slice(items, func(i, j int) bool {
		o1, _ := meta.Accessor(items[i])
		o2, _ := meta.Accessor(items[j])
		if o1 == nil || o2 == nil {
			return false
		}

		t1 := o1.GetCreationTimestamp()
		t2 := o2.GetCreationTimestamp()
		if t1.Equal(&t2) {
			return o1.GetName() < o2.GetName()
		}

		return t1.After(t2.Time)
	})

	user := c.MustGet("user").(model.User)
	filterItems := make([]runtime.Object, 0, len(items))
	for i := range items {
		obj, err := meta.Accessor(items[i])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access object metadata"})
			return zero, err
		}
		obj.SetManagedFields(nil)
		anno := obj.GetAnnotations()
		if anno != nil {
			delete(anno, common.KubectlAnnotation)
		}
		if h.Name() == string(common.Namespaces) && !rbac.CanAccessNamespace(user, cs.Name, obj.GetName()) {
			continue
		}
		if allNamespaces && obj.GetNamespace() != "" &&
			!rbac.CanAccess(user, h.Name(), string(common.VerbGet), cs.Name, obj.GetNamespace()) {
			continue
		}
		filterItems = append(filterItems, items[i])
	}
	_ = meta.SetList(objectList, filterItems)

	return objectList, nil
}

func (h *GenericResourceHandler[T, V]) List(c *gin.Context) {
	object, err := h.list(c)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, object)
}

func (h *GenericResourceHandler[T, V]) Search(c *gin.Context, q string, limit int64) ([]common.SearchResult, error) {
	if !h.enableSearch || q == "" {
		return nil, nil
	}
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)
	ctx := c.Request.Context()
	objectList := reflect.New(h.listType).Interface().(V)
	isLabelQuery := strings.Contains(q, ":") || strings.Contains(q, "=")

	// Sync engine fast path for plain name searches.
	if !isLabelQuery && syncengine.Handles(h.name) && syncengine.Active(cs.Name, h.name) {
		if rows, err := syncengine.SearchObjectRows(cs.Name, h.name, q, limit); err == nil {
			results := make([]common.SearchResult, 0, len(rows))
			for _, row := range rows {
				if h.Name() == string(common.Namespaces) {
					if !rbac.CanAccessNamespace(user, cs.Name, row.Name) {
						continue
					}
				} else {
					ns := row.Namespace
					if ns == "" {
						ns = common.AllNamespaces
					}
					if !rbac.CanAccess(user, h.Name(), string(common.VerbGet), cs.Name, ns) {
						continue
					}
				}
				results = append(results, common.SearchResult{
					ID:           row.UID,
					Name:         row.Name,
					Namespace:    row.Namespace,
					ResourceType: h.name,
					CreatedAt:    row.CreationTimestamp.String(),
				})
			}
			return results, nil
		}
	}

	var listOpts []client.ListOption
	if idx := strings.Index(q, ":"); idx > 0 {
		labelKey := strings.TrimSpace(q[:idx])
		labelValue := strings.TrimSpace(q[idx+1:])
		listOpts = append(listOpts, client.MatchingLabels{labelKey: labelValue})
	} else if idx := strings.Index(q, "="); idx > 0 {
		labelKey := strings.TrimSpace(q[:idx])
		labelValue := strings.TrimSpace(q[idx+1:])
		listOpts = append(listOpts, client.MatchingLabels{labelKey: labelValue})
	}
	if err := cs.K8sClient.List(ctx, objectList, listOpts...); err != nil {
		klog.Errorf("failed to list %s: %v", h.name, err)
		return nil, err
	}
	isLabelSearch := isLabelQuery
	items, err := meta.ExtractList(objectList)
	if err != nil {
		klog.Errorf("failed to extract items from list: %v", err)
		return nil, err
	}

	results := make([]common.SearchResult, 0, limit)

	for _, item := range items {
		obj, ok := item.(client.Object)
		if !ok {
			klog.Errorf("item is not a client.Object: %v", item)
			continue
		}
		if !isLabelSearch && !strings.Contains(strings.ToLower(obj.GetName()), strings.ToLower(q)) {
			continue
		}
		if h.Name() == string(common.Namespaces) {
			if !rbac.CanAccessNamespace(user, cs.Name, obj.GetName()) {
				continue
			}
		} else {
			namespace := obj.GetNamespace()
			if namespace == "" {
				namespace = common.AllNamespaces
			}
			if !rbac.CanAccess(user, h.Name(), string(common.VerbGet), cs.Name, namespace) {
				continue
			}
		}
		result := common.SearchResult{
			ID:           string(obj.GetUID()),
			Name:         obj.GetName(),
			Namespace:    obj.GetNamespace(),
			ResourceType: h.name,
			CreatedAt:    obj.GetCreationTimestamp().String(),
		}
		results = append(results, result)
		if limit > 0 && int64(len(results)) >= limit {
			break
		}
	}

	return results, nil
}
