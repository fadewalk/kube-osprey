# 资源同步

资源同步（Resource Sync）是一个可选引擎，把白名单内的 Kubernetes 资源类型从集群镜像到 KubeOsprey 数据库。开启后，资源列表、全局搜索和拓扑视图都从本地数据库读取，不再实时请求 API Server，让 KubeOsprey 在大集群下保持流畅。

引擎默认关闭；关闭时 KubeOsprey 始终直连 API Server。

## 启用资源同步

1. 打开 `Settings`。
2. 进入 `General`。
3. 打开 `Resource Sync` 开关。
4. 按需覆盖资源白名单（逗号分隔的复数名，例如 `pods,deployments,services`）。留空使用默认白名单：

   `pods, deployments, statefulsets, daemonsets, replicasets, jobs, services, ingresses, configmaps, persistentvolumeclaims, nodes, namespaces, events`

5. 保存设置。

::: warning
Secrets 被刻意排除在默认白名单之外。如果你自行添加 `secrets`，存储的对象 JSON 会包含 secret 数据本身，并落入 KubeOsprey 数据库。
:::

## 开启后的变化

- **资源列表**：白名单类型走数据库快路径。同步数据不新鲜或查询失败时，KubeOsprey 会自动回退到直连 API Server——行为不会中断，只是退回原有速度。
- **全局搜索**：纯名称查询走数据库索引，可跨所有已同步集群，跨集群搜索变快。
- **拓扑视图**：可用。基于同步数据沿 owner 引用（Pod ← ReplicaSet ← Deployment 等）绘制工作负载关系。
- 未进白名单的资源类型与之前完全一致，始终直连。

## 运维提示

- 引擎通过 Kubernetes watch（ListAndWatch，自动重连）保持数据新鲜；KubeOsprey 重启后会做一次全量 re-list。
- 同步对象存储在 KubeOsprey 数据库中。单二进制 SQLite 部署时，如果集群里高 churn 对象很多（例如大量 Job 或 Event），请关注数据库体积；增长过快就收窄白名单。
- 关闭引擎后 KubeOsprey 立即回到直连模式；已同步数据会保留直至清理。
