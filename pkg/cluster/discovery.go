package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/fadewalk/kube-osprey/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var discoveryLabels = []client.MatchingLabels{
	{
		"app.kubernetes.io/instance": "prometheus",
	},
	{
		"app.kubernetes.io/name": "prometheus",
	},
	{
		"app.kubernetes.io/part-of": "kube-prometheus-stack",
	},
	{
		"app.kubernetes.io/name": "vmsingle",
	},
}

func discoverServiceURL(kc *kube.K8sClient, matchLabels client.MatchingLabels, ports ...int32) string {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	var svcList corev1.ServiceList
	err := kc.List(ctx, &svcList, matchLabels)
	if err != nil {
		return ""
	}
	for _, svc := range svcList.Items {
		if svc.Spec.Type != corev1.ServiceTypeClusterIP && svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			continue
		}
		if svc.Spec.ClusterIP == corev1.ClusterIPNone {
			continue
		}
		for _, port := range svc.Spec.Ports {
			for _, wanted := range ports {
				if port.Port == wanted {
					return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", svc.Name, svc.Namespace, port.Port)
				}
			}
		}
	}
	return ""
}

func discoveryPrometheusURL(kc *kube.K8sClient) string {
	for _, matchLabels := range discoveryLabels {
		if url := discoverServiceURL(kc, matchLabels, 9090, 8428, 8429); url != "" {
			return url
		}
	}
	return ""
}

func discoveryAlertManagerURL(kc *kube.K8sClient) string {
	for _, matchLabels := range []client.MatchingLabels{
		{"app.kubernetes.io/name": "alertmanager"},
		{"app.kubernetes.io/component": "alertmanager"},
	} {
		if url := discoverServiceURL(kc, matchLabels, 9093); url != "" {
			return url
		}
	}
	return ""
}
