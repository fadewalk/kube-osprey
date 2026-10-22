# 告警中心

告警中心将每个集群中 [Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/) 触发的告警汇聚到统一视图，并支持在 KubeOsprey 内直接管理静默规则。

![告警中心](/screenshots/monitor.png)

## 前置条件

- 该集群已配置 Prometheus 监控，参见 [Prometheus 设置指南](../config/prometheus-setup)。
- 集群内存在可访问的 Alertmanager 端点。KubeOsprey 会自动发现以 `kube-prometheus-stack` 标准标签部署的 Alertmanager 服务；也可以在集群设置中显式填写地址。

## 功能

- **查看告警**：展示标签、注解、严重级别与触发时长。
- **创建静默**：对某条告警设置静默，在指定时间段内停止通知。
- **管理静默**：查看静默列表，维护窗口结束后手动过期静默。
- **查看 Alertmanager 状态**：确认集成是否健康。

访问遵循集群级 RBAC：无集群访问权限的用户看不到对应告警。

## API

告警中心由以下集群级接口支撑：

| 方法     | 路径                                    | 用途             |
| -------- | --------------------------------------- | ---------------- |
| `GET`    | `/api/v1/alertmanager/alerts`           | 查看活跃告警     |
| `GET`    | `/api/v1/alertmanager/silences`         | 查看静默列表     |
| `POST`   | `/api/v1/alertmanager/silences`         | 创建静默         |
| `DELETE` | `/api/v1/alertmanager/silences/:silenceId` | 过期静默     |
| `GET`    | `/api/v1/alertmanager/status`           | Alertmanager 状态 |

::: tip 延伸阅读
告警中心与[健康诊断](./health-diagnostics)搭配使用效果最好：诊断告诉你集群内部什么在劣化，告警告诉你监控系统已经发现了什么。
:::
