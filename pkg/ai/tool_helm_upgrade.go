package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/helmutil"
	pkgmodel "github.com/fadewalk/kube-osprey/pkg/model"
)

// executeHelmUpgradeRelease upgrades a helm release from a change-plan step
// or a CI webhook. The cluster comes from the step, not the args.
func executeHelmUpgradeRelease(ctx context.Context, cs *cluster.ClientSet, user pkgmodel.User, args map[string]interface{}) (string, error) {
	namespace, err := getRequiredString(args, "namespace")
	if err != nil {
		return "", err
	}
	releaseName, err := getRequiredString(args, "release")
	if err != nil {
		return "", err
	}

	str := func(key string) string {
		v, _ := args[key].(string)
		return strings.TrimSpace(v)
	}
	req := helmutil.UpgradeRequest{
		ChartURL:       str("chartUrl"),
		ChartVersion:   str("chartVersion"),
		RepositoryName: str("repositoryName"),
		Source:         str("source"),
		Description:    str("description"),
	}
	if values, ok := args["values"].(map[string]interface{}); ok {
		req.Values = values
	}
	if b, ok := args["forceConflicts"].(bool); ok {
		req.ForceConflicts = b
	}
	if b, ok := args["rollbackOnFailure"].(bool); ok {
		req.RollbackOnFailure = b
	}
	if b, ok := args["wait"].(bool); ok {
		req.Wait = b
	}

	result, _, err := helmutil.RunUpgrade(ctx, cs, user, namespace, releaseName, req, false)
	if err != nil {
		return "", err
	}
	revision := 0
	if result.Release != nil {
		revision = result.Release.Version
	}
	return fmt.Sprintf("helm release %s/%s upgraded to revision %d", namespace, releaseName, revision), nil
}
