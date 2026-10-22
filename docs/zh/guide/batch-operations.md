# 批量运维

批量运维让你在列表页上一次对多个工作负载或节点执行同一操作，而不必逐个资源重复编辑。

## 工作负载

在 Deployment、StatefulSet、DaemonSet 的列表页可以多选工作负载并执行：

- **更新镜像**：为所有选中工作负载设置新容器镜像，例如协调发布一个新应用版本。
- **重启**：触发所有选中工作负载的滚动重启。
- **伸缩**：调整选中工作负载的副本数。

::: warning
每次操作都要求对目标 Kind 和命名空间拥有 `update` 权限。KubeOsprey 会在执行前逐个工作负载检查 RBAC，且每个变更都会记录到审计日志和[资源历史](./resource-history)。
:::

## 节点

在节点列表页可以对选中节点执行 **cordon**、**uncordon** 和 **drain**，这是维护窗口前的标准准备动作。

## API

工作负载操作同样开放给自动化调用：

| 方法   | 路径                                    | 适用对象                                      |
| ------ | --------------------------------------- | --------------------------------------------- |
| `POST` | `/api/v1/workload-ops/update-image`     | deployments、statefulsets、daemonsets         |
| `POST` | `/api/v1/workload-ops/restart`          | deployments、statefulsets、daemonsets         |
