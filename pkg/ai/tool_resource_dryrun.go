package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zxh326/kite/pkg/cluster"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DryRunMutation performs a server-side dry-run of a mutation tool call and
// returns before/after snapshots without recording history or changing state.
// delete has no meaningful dry-run; it is reported as unsupported with the
// current object captured as the "before" snapshot.
func DryRunMutation(ctx context.Context, cs *cluster.ClientSet, toolName string, args map[string]any) (before, after map[string]any, supported bool, err error) {
	if cs == nil {
		return nil, nil, false, fmt.Errorf("cluster client is required")
	}
	switch toolName {
	case "create_resource", "apply_resource", "update_resource":
		return dryRunObjectMutation(ctx, cs, args)
	case "patch_resource":
		return dryRunPatch(ctx, cs, args)
	case "delete_resource":
		beforeObj, err := currentObject(ctx, cs, args)
		if apierrors.IsNotFound(err) {
			// The target does not exist yet — common when an earlier plan
			// step creates it. Nothing to snapshot; report as unsupported
			// so the plan can still proceed (apply fails honestly if the
			// object is absent at execution time).
			return nil, nil, false, nil
		}
		if err != nil {
			return nil, nil, false, err
		}
		before, err := objectToMap(beforeObj)
		return before, nil, false, err
	case "helm_upgrade_release":
		// The webhook's own dryRun payload already previews the upgrade;
		// plan steps record it as dry-run unsupported.
		return nil, nil, false, nil
	default:
		return nil, nil, false, fmt.Errorf("tool %q does not support dry-run", toolName)
	}
}

func dryRunObjectMutation(ctx context.Context, cs *cluster.ClientSet, args map[string]any) (map[string]any, map[string]any, bool, error) {
	obj, err := parseResourceYAML(args)
	if err != nil {
		return nil, nil, false, err
	}
	resource := resolveResourceInfoForObject(ctx, cs, obj)
	obj.SetNamespace(normalizeNamespace(resource, obj.GetNamespace()))

	existing := obj.DeepCopy()
	err = cs.K8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}, existing)

	beforeMap, err := snapshotOr(existing, err)
	if err != nil {
		return nil, nil, false, err
	}

	dry := obj.DeepCopy()
	var dryErr error
	if beforeMap != nil {
		obj.SetResourceVersion(existing.GetResourceVersion())
		dry = obj.DeepCopy()
		dryErr = cs.K8sClient.Update(ctx, dry, client.DryRunAll)
	} else {
		dryErr = cs.K8sClient.Create(ctx, dry, client.DryRunAll)
	}
	if dryErr != nil {
		return beforeMap, nil, false, dryErr
	}
	afterMap, err := objectToMap(dry)
	if err != nil {
		return beforeMap, nil, false, err
	}
	return beforeMap, afterMap, true, nil
}

func dryRunPatch(ctx context.Context, cs *cluster.ClientSet, args map[string]any) (map[string]any, map[string]any, bool, error) {
	kind, err := getRequiredString(args, "kind")
	if err != nil {
		return nil, nil, false, err
	}
	name, err := getRequiredString(args, "name")
	if err != nil {
		return nil, nil, false, err
	}
	namespace, _ := args["namespace"].(string)
	patchStr, err := getRequiredString(args, "patch")
	if err != nil {
		return nil, nil, false, err
	}
	if !json.Valid([]byte(patchStr)) {
		return nil, nil, false, fmt.Errorf("patch must be valid JSON")
	}
	resource := resolveResourceInfo(ctx, cs, kind)
	obj := buildObjectForResource(resource)
	key := k8stypes.NamespacedName{Name: name, Namespace: normalizeNamespace(resource, namespace)}
	if err := cs.K8sClient.Get(ctx, key, obj); err != nil {
		return nil, nil, false, err
	}
	beforeMap, err := objectToMap(obj)
	if err != nil {
		return nil, nil, false, err
	}

	patchType := k8stypes.StrategicMergePatchType
	switch value, _ := args["patch_type"].(string); value {
	case "merge":
		patchType = k8stypes.MergePatchType
	case "json":
		patchType = k8stypes.JSONPatchType
	}
	dry := obj.DeepCopy()
	if err := cs.K8sClient.Patch(ctx, dry, client.RawPatch(patchType, []byte(patchStr)), client.DryRunAll); err != nil {
		return beforeMap, nil, false, err
	}
	afterMap, err := objectToMap(dry)
	if err != nil {
		return beforeMap, nil, false, err
	}
	return beforeMap, afterMap, true, nil
}

// currentObject resolves the live object a mutation targets, for before
// snapshots; delete only needs this snapshot.
func currentObject(ctx context.Context, cs *cluster.ClientSet, args map[string]any) (*unstructured.Unstructured, error) {
	obj, err := parseResourceYAML(args)
	if err == nil {
		resource := resolveResourceInfoForObject(ctx, cs, obj)
		obj.SetNamespace(normalizeNamespace(resource, obj.GetNamespace()))
		err = cs.K8sClient.Get(ctx, k8stypes.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		}, obj)
		return obj, err
	}
	// delete/patch style args carry kind/name/namespace instead of yaml.
	kind, kindErr := getRequiredString(args, "kind")
	if kindErr != nil {
		return nil, err
	}
	name, nameErr := getRequiredString(args, "name")
	if nameErr != nil {
		return nil, err
	}
	namespace, _ := args["namespace"].(string)
	resource := resolveResourceInfo(ctx, cs, kind)
	obj = buildObjectForResource(resource)
	if getErr := cs.K8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      name,
		Namespace: normalizeNamespace(resource, namespace),
	}, obj); getErr != nil {
		return nil, getErr
	}
	return obj, nil
}

func snapshotOr(obj *unstructured.Unstructured, getErr error) (map[string]any, error) {
	if apierrors.IsNotFound(getErr) {
		return nil, nil
	}
	if getErr != nil {
		return nil, getErr
	}
	return objectToMap(obj)
}

func objectToMap(obj *unstructured.Unstructured) (map[string]any, error) {
	if obj == nil {
		return nil, nil
	}
	clean := obj.DeepCopy()
	unstructured.RemoveNestedField(clean.Object, "metadata", "managedFields")
	return clean.Object, nil
}
