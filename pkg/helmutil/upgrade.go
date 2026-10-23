package helmutil

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
	release "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/klog/v2"
)

const helmUpgradeTimeout = 5 * time.Minute

// UpgradeRequest describes a helm release upgrade. It carries the fields the
// console upgrade action and the CI webhook accept.
type UpgradeRequest struct {
	ChartURL string
	// ChartVersion upgrades to that version from the release's repository
	// catalog. Ignored when ChartURL is set; empty keeps the current chart.
	ChartVersion      string
	RepositoryName    string
	Source            string
	Values            map[string]interface{}
	Description       string
	ForceConflicts    bool
	RollbackOnFailure bool
	Wait              bool
}

// RunResult carries the release state before and after the upgrade.
type RunResult struct {
	Current *release.Release
	Release *release.Release
}

// RunUpgrade performs a helm release upgrade as user. The returned status is
// HTTP-flavored so handlers can pass it through verbatim on failure.
func RunUpgrade(ctx context.Context, cs *cluster.ClientSet, user model.User, namespace, name string, req UpgradeRequest, dryRun bool) (result RunResult, status int, err error) {
	cfg, err := NewActionConfig(cs.K8sClient.Configuration, StorageNamespace(namespace))
	if err != nil {
		return RunResult{}, http.StatusInternalServerError, err
	}
	current, err := GetRelease(cfg, name)
	if err != nil {
		return RunResult{}, http.StatusInternalServerError, err
	}
	if current.Chart == nil {
		return RunResult{}, http.StatusInternalServerError, fmt.Errorf("helm release chart is missing")
	}
	result.Current = current
	if !dryRun {
		defer func() {
			RecordReleaseHistory(cs.Name, user.ID, "manual", "upgrade", name, namespace, current, result.Release, err == nil, err)
		}()
	}

	chartToUpgrade := current.Chart
	if strings.TrimSpace(req.ChartURL) != "" {
		req.ChartURL = strings.TrimSpace(req.ChartURL)
		repository, err := ResolveChartRepository(
			strings.TrimSpace(req.RepositoryName),
			strings.TrimSpace(req.Source),
		)
		if err != nil {
			return RunResult{}, http.StatusBadRequest, fmt.Errorf("repository not found")
		}
		chartToUpgrade, err = LoadArchive(req.ChartURL, repository)
		if err != nil {
			return RunResult{}, http.StatusBadRequest, err
		}
	} else if strings.TrimSpace(req.ChartVersion) != "" {
		repository, err := ResolveChartRepository(
			strings.TrimSpace(req.RepositoryName),
			strings.TrimSpace(req.Source),
		)
		if err != nil {
			return RunResult{}, http.StatusBadRequest, fmt.Errorf("repository not found")
		}
		if current.Chart == nil || current.Chart.Metadata == nil || current.Chart.Metadata.Name == "" {
			return RunResult{}, http.StatusBadRequest, fmt.Errorf("current release chart metadata is missing, cannot resolve chart version")
		}
		chartToUpgrade, err = loadChartVersion(repository, current.Chart.Metadata.Name, strings.TrimSpace(req.ChartVersion))
		if err != nil {
			return RunResult{}, http.StatusBadRequest, err
		}
	}

	values := req.Values
	if values == nil {
		values = map[string]interface{}{}
	}
	description := req.Description
	if description == "" {
		description = "Dry run upgrade requested from KubeOsprey"
		if !dryRun {
			description = "Upgrade requested from KubeOsprey"
		}
	}

	rel, err := UpgradeRelease(ctx, cfg, name, chartToUpgrade, values, UpgradeReleaseOptions{
		Namespace:         namespace,
		Timeout:           helmUpgradeTimeout,
		ReuseValues:       req.Values == nil,
		Description:       description,
		ForceConflicts:    req.ForceConflicts,
		RollbackOnFailure: req.RollbackOnFailure,
		DryRun:            dryRun,
		Wait:              req.Wait,
	})
	if err != nil {
		klog.Errorf("Failed to upgrade helm release %s/%s: %v", namespace, name, err)
		return RunResult{}, http.StatusInternalServerError, err
	}
	result.Release = rel
	return result, http.StatusOK, nil
}
