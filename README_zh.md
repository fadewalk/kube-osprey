<div align="center">

[English](./README.md) · **简体中文**

<img src="./docs/assets/logo.svg" alt="KubeOsprey Logo" width="128" height="128">

# KubeOsprey

**所有集群，一个控制台。**

KubeOsprey 是一个单二进制 Kubernetes 工作空间，面向多集群运维、可观测性、访问控制与 AI 辅助排障。

[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg?style=flat-square)](LICENSE)

[![Upstream](https://img.shields.io/badge/fork%20of-Kite-blue?style=flat-square)](https://github.com/kite-org/kite)

</div>

<img width="1586" height="1167" alt="image" src="https://github.com/user-attachments/assets/a88a63b7-5b71-444d-8d98-66f147a68ef7" />

> [!NOTE]
> KubeOsprey 是 [Kite](https://github.com/kite-org/kite)（Apache-2.0）的独立维护 fork：持续吸收上游有价值的更新，但独立发展、不向上游回传改动。代码层标识（Go module 路径、`KITE_*` 环境变量、`charts/kite`）刻意保留原名，以保证上游变更可以持续干净地合并。KubeOsprey 是独立的社区项目，与 Kubernetes 项目及 CNCF 无隶属或背书关系。

## 为什么选择 KubeOsprey

### 可观测

- 通过各集群独立配置的 Prometheus 查看实时 CPU、内存和网络指标。
- 实时查看并过滤 Pod 日志，在同一上下文中检查事件、状态和关联资源。

### 运维

- 跨多个集群管理 Kubernetes 核心资源、CRD 和 Helm Release。
- 在线编辑 YAML、扩缩容或重启工作负载，并使用面向 Pod、Node 和 kubectl 的 Web 终端。
- 通过全局搜索快速定位资源，或使用内置 Kube Proxy 直接访问 Pod 和 Service。

### 治理

- 支持 OAuth、LDAP、本地账户、MFA 和 Passkey 登录。
- 通过 RBAC 和集群访问权限控制操作范围，并使用审计日志追踪资源变更。

### AI 辅助

- 使用内置 AI 助手检查资源、日志和 Prometheus 指标。
- 写操作执行前需要用户确认，且 AI 助手遵循当前用户的 RBAC 权限。
- Claude Code、Cursor 等 MCP 客户端可接入内置 [MCP 服务器](https://fadewalk.github.io/kube-osprey/zh/guide/mcp-server)；Agent 的写操作会以[变更计划](https://fadewalk.github.io/kube-osprey/zh/guide/change-plans)形式挂起，经人工批准后才会执行。

## 快速开始

### 从源码构建

构建 KubeOsprey 需要 Go 1.27、Node.js `^20.19.0` 或 `>=22.12.0`、pnpm 和 Make。

```bash
git clone https://github.com/fadewalk/kube-osprey.git
cd kube-osprey
make deps
make build
./kube-osprey
```

### 安装发行版

每个 tag 版本都会在 [Releases](https://github.com/fadewalk/kube-osprey/releases) 页面发布容器镜像、OCI Helm Chart 以及 Linux / macOS / Windows 独立二进制。Chart 沿用上游名称 `kite`，因此 `KITE_*` 环境变量与既有部署保持兼容；`fullnameOverride` 让部署出来的资源统一命名为 `kube-osprey`。

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
> Chart 默认值仅适合快速体验。用于生产环境前，请启用持久化存储或配置外部数据库、显式固定 `encryptKey` 而不要依赖自动生成的值，并检查 Chart 创建的集群级 RBAC 权限。KubeOsprey 会拒绝使用上游默认加密密钥或短于 32 个字符的密钥启动。详见上游[安装指南](https://fadewalk.github.io/kube-osprey/zh/guide/installation)和 [Chart Values](https://fadewalk.github.io/kube-osprey/zh/config/chart-values)。

## 文档

| 主题             | 上游指南                                                                                                                        |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| 安装与访问       | [安装指南](https://fadewalk.github.io/kube-osprey/zh/guide/installation)                                                                            |
| 用户、认证和权限 | [用户管理](https://fadewalk.github.io/kube-osprey/zh/config/user-management) · [RBAC](https://fadewalk.github.io/kube-osprey/zh/config/rbac-config)                     |
| 监控             | [Prometheus 配置](https://fadewalk.github.io/kube-osprey/zh/config/prometheus-setup) · [告警中心](https://fadewalk.github.io/kube-osprey/zh/guide/alert-center) · [健康诊断](https://fadewalk.github.io/kube-osprey/zh/guide/health-diagnostics)                                  |
| 运维             | [Helm 管理](https://fadewalk.github.io/kube-osprey/zh/guide/helm-management) · [Kite Cluster Agent](https://fadewalk.github.io/kube-osprey/zh/guide/kite-cluster-agent) · [批量运维](https://fadewalk.github.io/kube-osprey/zh/guide/batch-operations)                             |
| 变更管理         | [变更计划](https://fadewalk.github.io/kube-osprey/zh/guide/change-plans) · [通知渠道](https://fadewalk.github.io/kube-osprey/zh/config/notification-channels)                                                                                            |
| 规模与搜索       | [资源同步](https://fadewalk.github.io/kube-osprey/zh/guide/resource-sync) · [全局搜索](https://fadewalk.github.io/kube-osprey/zh/guide/global-search)                                                                                                                   |
| AI               | [AI 助手](https://fadewalk.github.io/kube-osprey/zh/guide/ai-assistant) · [MCP 服务器](https://fadewalk.github.io/kube-osprey/zh/guide/mcp-server)                                                                             |
| API              | [API 文档](https://fadewalk.github.io/kube-osprey/zh/api/authentication)                                                                            |

## 许可证

KubeOsprey 基于 [Apache License 2.0](LICENSE) 发布，许可继承自 Kite。
