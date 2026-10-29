package scheduler

import (
	"context"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
)

func Start(ctx context.Context, cm *cluster.ClusterManager) {
	manager := NewManager()

	registerHelmReleaseAutoUpgradeExecutor(manager, cm)
	manager.Start(ctx)
}
