<div align="center">

<img src="./docs/assets/logo.svg" alt="KubeOsprey logo" width="128" height="128">

# KubeOsprey

**One console for all your clusters.**

KubeOsprey is a single-binary Kubernetes workspace for multi-cluster operations, observability, access control, and AI-assisted troubleshooting.

[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg?style=flat-square)](LICENSE)

[![Upstream](https://img.shields.io/badge/fork%20of-Kite-blue?style=flat-square)](https://github.com/kite-org/kite)

**English** · [中文](./README_zh.md)

</div>

<img width="1586" height="1167" alt="image" src="https://github.com/user-attachments/assets/5710204d-5d34-44af-85dc-3b436e205c12" />

> [!NOTE]
> KubeOsprey is an independently maintained fork of [Kite](https://github.com/kite-org/kite) (Apache-2.0). It pulls useful upstream improvements but is developed separately and does not contribute changes back. Code-level identifiers (the Go module path and `KITE_*` environment variables) intentionally keep their original names so upstream changes keep merging cleanly. KubeOsprey is an independent community project and is not affiliated with or endorsed by the Kubernetes project or the CNCF.

## Why KubeOsprey

### Observe

- Monitor CPU, memory, and network usage with per-cluster Prometheus integrations.
- Stream and filter pod logs, then inspect events, conditions, and related resources in context.

### Operate

- Manage core Kubernetes resources, CRDs, and Helm releases across multiple clusters.
- Edit live YAML, scale or restart workloads, and use browser-based pod, node, and kubectl terminals.
- Find resources with global search and access pods or services through the built-in Kube Proxy.

### Govern

- Support OAuth, LDAP, local accounts, MFA, and passkey login.
- Control access with RBAC and per-cluster permissions, backed by audit logs for resource changes.

### Work with AI

- Inspect resources, logs, and Prometheus metrics with the built-in AI assistant.
- Review and confirm write operations before execution; the assistant respects the current user's RBAC permissions.

## Quick Start

### Build from Source

Building KubeOsprey requires Go 1.26, Node.js `^20.19.0` or `>=22.12.0`, pnpm, and Make.

```bash
git clone https://github.com/fadewalk/kube-osprey.git
cd kube-osprey
make deps
make build
./kube-osprey
```

### Upstream Releases (Kite)

The upstream project publishes ready-made Helm charts, Docker images, and manifests under the Kite name:

```bash
helm install kite oci://ghcr.io/kite-org/charts/kite \
  --namespace kite-system \
  --create-namespace
```

```bash
mkdir -p data
docker run -d --name kite \
  -p 8080:8080 \
  -v "$(pwd)/data:/data" \
  -e DB_DSN=/data/db.sqlite \
  ghcr.io/kite-org/kite:latest
```

> [!IMPORTANT]
> The default chart values are intended for evaluation. Before using it in production, enable persistent storage or configure an external database, replace the default encryption key, and review the chart's cluster-wide RBAC permissions. See the upstream [installation guide](https://kitehq.dev/guide/installation) and [chart values](https://kitehq.dev/config/chart-values).

## Documentation

| Topic                                  | Upstream guide                                                                                                                  |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Installation and exposure              | [Installation](https://kitehq.dev/guide/installation)                                                                           |
| Users, authentication, and permissions | [User management](https://kitehq.dev/config/user-management) · [RBAC](https://kitehq.dev/config/rbac-config)                    |
| Monitoring                             | [Prometheus setup](https://kitehq.dev/config/prometheus-setup)                                                                  |
| Operations                             | [Helm management](https://kitehq.dev/guide/helm-management) · [Kite Cluster Agent](https://kitehq.dev/guide/kite-cluster-agent) |
| AI                                     | [AI assistant](https://kitehq.dev/guide/ai-assistant)                                                                           |
| API                                    | [API documentation](https://kitehq.dev/api/authentication)                                                                      |

## License

KubeOsprey is licensed under the [Apache License 2.0](LICENSE), inherited from Kite.
