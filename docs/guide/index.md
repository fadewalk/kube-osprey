# What is KubeOsprey?

KubeOsprey is a lightweight, open-source Kubernetes workspace for platform teams. It brings multi-cluster operations, observability, access control, and AI-assisted troubleshooting into one interface.

![KubeOsprey dashboard showing cluster health, workloads, events, and resource usage](/screenshots/overview.png)

## Start here

::: tip Fastest evaluation path
Install KubeOsprey with Helm, use port forwarding for the first visit, then connect the in-cluster Kubernetes API. You can complete the initial setup without preparing a kubeconfig.
:::

1. [Install KubeOsprey](./installation) and open the dashboard.
2. [Set up users](/config/user-management) and [RBAC](/config/rbac-config) before inviting your team.
3. [Connect Prometheus](/config/prometheus-setup) when you are ready to add cluster metrics.

## Find the right guide

- **See how the pieces fit together:** [Architecture](./architecture)
- **Find any resource quickly:** [Global Search](./global-search)
- **Understand a workload and its dependencies:** [Related Resources](./related-resources)
- **Investigate a running Pod:** [Logs](./logs) and [Web Terminal](./web-terminal)
- **Stream logs from many pods at once:** [Log Center](./log-center)
- **Track cluster health and usage:** [Monitoring](./monitoring)
- **Triage alerts and silences:** [Alert Center](./alert-center)
- **Assess cluster health at a glance:** [Health Diagnostics](./health-diagnostics)
- **Edit many workloads at once:** [Batch Operations](./batch-operations)
- **Install and manage releases:** [Helm Management](./helm-management)
- **Gate sensitive changes behind approval:** [Change Plans](./change-plans)
- **Review resource changes:** [Resource History](./resource-history)
- **Troubleshoot with an AI agent:** [AI Assistant](./ai-assistant)
- **Connect private or remote clusters:** [KubeOsprey Cluster Agent](./cluster-agent)

## Platform at a glance

### Observe

Use real-time CPU, memory, and network charts, live Pod logs, events, and resource relationships to move from a symptom to its context. The [Log Center](./log-center) aggregates logs across pods, the [Alert Center](./alert-center) surfaces Alertmanager alerts, and [Health Diagnostics](./health-diagnostics) scores each cluster's health.

### Operate

Create and edit Kubernetes resources, manage Helm releases, open Pod and Node terminals, run kubectl, and access workloads through Kube Proxy. [Batch Operations](./batch-operations) applies image updates, restarts, and scaling to many workloads at once, and [Change Plans](./change-plans) puts AI and CI-triggered changes behind a human approval gate with rollback support.

### Govern

Manage team access with OAuth, MFA, passkeys, users, RBAC, role mappings, and audit logs across multiple clusters. Role assignments support expiry, and password policies with forced password change keep local accounts safe.

## Where KubeOsprey fits

Headlamp and Kubernetes Dashboard are strong tools for inspecting and operating individual clusters. KubeOsprey adds a shared workspace for multi-cluster operations, built-in team governance, observability, and AI-assisted workflows.

If you are evaluating KubeOsprey, continue with the [installation guide](./installation).
