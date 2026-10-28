<div align="center">

**English** · [简体中文](./README_zh.md)

<img src="./docs/assets/logo.svg" alt="KubeOsprey logo" width="128" height="128">

# KubeOsprey

**One console for all your clusters.**

KubeOsprey is a single-binary Kubernetes workspace for multi-cluster operations, observability, access control, and AI-assisted troubleshooting.

[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg?style=flat-square)](LICENSE)

[![Upstream](https://img.shields.io/badge/fork%20of-Kite-blue?style=flat-square)](https://github.com/kite-org/kite)

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
- Connect Claude Code, Cursor, or any MCP client to the built-in [MCP server](https://fadewalk.github.io/kube-osprey/guide/mcp-server); agent write operations are held as [change plans](https://fadewalk.github.io/kube-osprey/guide/change-plans) until a human approves them.

## Quick Start

### Build from Source

Building KubeOsprey requires Go 1.27, Node.js `^20.19.0` or `>=22.12.0`, pnpm, and Make.

```bash
git clone https://github.com/fadewalk/kube-osprey.git
cd kube-osprey
make deps
make build
./kube-osprey
```

### Install a Release

Every tagged release publishes a container image, an OCI Helm chart, and standalone binaries (Linux, macOS, Windows) on the [Releases](https://github.com/fadewalk/kube-osprey/releases) page. The chart keeps the upstream name `kite`, so `KITE_*` environment variables stay compatible with existing deployments; `fullnameOverride` keeps the deployed resources named `kube-osprey`.

```bash
helm install kube-osprey oci://ghcr.io/fadewalk/charts/kite \
  --namespace kube-osprey-system \
  --create-namespace \
  --set fullnameOverride=kube-osprey
```

```bash
mkdir -p data
[ -f data/encrypt.key ] || openssl rand -hex 32 > data/encrypt.key
docker run -d --name kube-osprey \
  -p 8080:8080 \
  -v "$(pwd)/data:/data" \
  -e DB_DSN=/data/db.sqlite \
  -e KITE_ENCRYPT_KEY="$(cat data/encrypt.key)" \
  ghcr.io/fadewalk/kube-osprey:latest
```

> [!IMPORTANT]
> The default chart values are intended for evaluation. Before using it in production, enable persistent storage or configure an external database, pin `encryptKey` instead of relying on the generated one, and review the chart's cluster-wide RBAC permissions. KubeOsprey refuses to start with the upstream default encryption key or one shorter than 32 characters. See the upstream [installation guide](https://fadewalk.github.io/kube-osprey/guide/installation) and [chart values](https://fadewalk.github.io/kube-osprey/config/chart-values).

## Documentation

| Topic                                  | Upstream guide                                                                                                                  |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Installation and exposure              | [Installation](https://fadewalk.github.io/kube-osprey/guide/installation)                                                                           |
| Users, authentication, and permissions | [User management](https://fadewalk.github.io/kube-osprey/config/user-management) · [RBAC](https://fadewalk.github.io/kube-osprey/config/rbac-config)                    |
| Monitoring                             | [Prometheus setup](https://fadewalk.github.io/kube-osprey/config/prometheus-setup) · [Alert Center](https://fadewalk.github.io/kube-osprey/guide/alert-center) · [Health diagnostics](https://fadewalk.github.io/kube-osprey/guide/health-diagnostics)                                 |
| Operations                             | [Helm management](https://fadewalk.github.io/kube-osprey/guide/helm-management) · [Kite Cluster Agent](https://fadewalk.github.io/kube-osprey/guide/kite-cluster-agent) · [Batch operations](https://fadewalk.github.io/kube-osprey/guide/batch-operations)                                            |
| Change management                      | [Change plans](https://fadewalk.github.io/kube-osprey/guide/change-plans) · [Notification channels](https://fadewalk.github.io/kube-osprey/config/notification-channels)                                                                                              |
| Scale & search                        | [Resource sync](https://fadewalk.github.io/kube-osprey/guide/resource-sync) · [Global search](https://fadewalk.github.io/kube-osprey/guide/global-search)                                                                                                            |
| AI                                     | [AI assistant](https://fadewalk.github.io/kube-osprey/guide/ai-assistant) · [MCP server](https://fadewalk.github.io/kube-osprey/guide/mcp-server)                                                                           |
| API                                    | [API documentation](https://fadewalk.github.io/kube-osprey/api/authentication)                                                                      |

## License

KubeOsprey is licensed under the [Apache License 2.0](LICENSE), inherited from Kite.
