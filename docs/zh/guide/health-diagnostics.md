# 健康诊断

健康诊断对集群运行一组内置检查，把结果汇总为一个健康评分和可操作的风险项列表，回答"这个集群现在健康吗"，而不需要自己手写查询。

![健康诊断](/screenshots/monitor.png)

## 报告内容

- **健康评分**：由检查结果推导出的 0–100 总分。
- **风险项**：按四个类别分组，每项带严重级别（`critical`、`warning`、`info`）、发生次数与样例：
  - `nodes` — NotReady 节点、资源压力等节点层问题。
  - `workloads` — 异常或频繁重启的 Pod、不可用副本。
  - `storage` — PVC 绑定与容量问题。
  - `platform` — 控制面与平台组件健康。
- **控制面组件**：API Server、Scheduler、Controller Manager、etcd 的 ready/total 状态。
- **资源消耗 Top 榜**：CPU 与内存用量最高的节点和 Pod。该部分仅在集群启用 [Prometheus](../config/prometheus-setup) 时展示。

## 多集群总览

总览页聚合你可访问的全部集群的健康状况，一眼定位集群中出问题的那一个。

## API

| 方法  | 路径                            | 用途                       |
| ----- | ------------------------------- | -------------------------- |
| `GET` | `/api/v1/health-report`         | 单集群健康报告             |
| `GET` | `/api/v1/clusters/overview`     | 多集群聚合总览             |

::: tip
健康检查在每次请求时实时计算、不落库——页面加载看到的就是集群当前状态。
:::
