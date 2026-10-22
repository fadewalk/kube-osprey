# 什么是 KubeOsprey？

KubeOsprey 是面向平台团队的轻量、开源 Kubernetes 工作空间，将多集群运维、可观测性、访问控制和 AI 辅助排障整合到同一个界面中。

![KubeOsprey 仪表盘展示集群健康状态、工作负载、事件和资源用量](/screenshots/overview.png)

## 从这里开始

::: tip 最快体验路径
使用 Helm 安装 KubeOsprey，首次访问时通过端口转发打开界面，然后连接集群内的 Kubernetes API。整个初始设置不需要提前准备 kubeconfig。
:::

1. [安装 KubeOsprey](./installation) 并打开仪表盘。
2. 邀请团队前，先配置[用户](/zh/config/user-management)和 [RBAC](/zh/config/rbac-config)。
3. 需要集群指标时，再[连接 Prometheus](/zh/config/prometheus-setup)。

## 找到对应指南

- **快速找到任意资源：** [全局搜索](./global-search)
- **了解工作负载及其依赖关系：** [相关资源](./related-resources)
- **排查正在运行的 Pod：** [日志](./logs)与 [Web 终端](./web-terminal)
- **一次聚合多个 Pod 的日志：** [日志中心](./log-center)
- **查看集群健康状态和用量：** [监控](./monitoring)
- **处理告警与静默：** [告警中心](./alert-center)
- **一眼评估集群健康度：** [健康诊断](./health-diagnostics)
- **批量编辑多个工作负载：** [批量运维](./batch-operations)
- **安装和管理应用：** [Helm 管理](./helm-management)
- **为敏感变更增加审批闸门：** [变更计划](./change-plans)
- **审查资源变更：** [资源历史](./resource-history)
- **使用 AI Agent 辅助排障：** [AI 助手](./ai-assistant)
- **连接私有或远程集群：** [KubeOsprey Cluster Agent](./kite-cluster-agent)

## 平台能力概览

### 观察

通过实时 CPU、内存和网络图表、Pod 日志、事件与资源关系，从异常现象快速定位所需上下文。[日志中心](./log-center)聚合多 Pod 日志，[告警中心](./alert-center)呈现 Alertmanager 告警，[健康诊断](./health-diagnostics)为每个集群给出健康评分。

### 运维

创建和编辑 Kubernetes 资源、管理 Helm Release、打开 Pod 与 Node 终端、运行 kubectl，并通过 Kube Proxy 访问工作负载。[批量运维](./batch-operations)一次对多个工作负载执行改镜像、重启与伸缩，[变更计划](./change-plans)为 AI 与 CI 触发的变更加上人工审批闸门并支持回滚。

### 治理

使用 OAuth、MFA、Passkey、用户管理、RBAC、角色映射和审计日志，统一管理多集群中的团队访问。角色授权支持过期时间，密码策略与强制改密保障本地账号安全。

## KubeOsprey 适合什么场景

Headlamp 和 Kubernetes Dashboard 都是优秀的单集群资源查看与操作工具。KubeOsprey 在此基础上增加了共享的多集群工作空间、团队治理、可观测性和 AI 辅助工作流。

如果你正在评估 KubeOsprey，请继续阅读[安装指南](./installation)。
