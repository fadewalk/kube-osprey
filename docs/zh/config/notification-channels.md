# 通知渠道

通知渠道让 KubeOsprey 把变更计划（Change Plan）生命周期事件推送到团队的聊天工具。管理员在 **Settings → Notification Channels** 中配置渠道，每个渠道订阅自己关心的事件。

![通知渠道设置页](/screenshots/notification-channels.png)

## 支持的渠道类型

| 类型       | 协议                                                                 | 说明                                   |
| ---------- | -------------------------------------------------------------------- | -------------------------------------- |
| `dingtalk` | 钉钉群机器人，Markdown 消息体                                        | 可通过 secret 启用 HMAC-SHA256 加签     |
| `wecom`    | 企业微信群机器人，Markdown 消息体                                    | 超过 4096 字节的消息会被截断            |
| `feishu`   | 飞书机器人，文本消息体                                               | 可通过 secret 启用时间戳加签            |
| `slack`    | Slack incoming webhook，`{"text": ...}`                              |                                        |
| `webhook`  | 通用接收端，JSON `{"title": ..., "text": ...}`                       | 指向任意 HTTP 端点即可                  |

## 事件

| 事件            | 含义                       |
| --------------- | -------------------------- |
| `plan_pending`  | 变更计划待审批             |
| `plan_applied`  | 变更计划执行成功           |
| `plan_failed`   | 变更计划执行失败           |
| `alert_firing`  | 集群告警开始触发           |
| `event_warning` | 出现新的 kubernetes 警告事件 |

每个渠道把订阅的事件以逗号分隔存储，一个渠道可以同时接收多种事件。

## 集群告警与警告事件

KubeOsprey 每分钟轮询一次所有已连接集群，把新出现的 Alertmanager 告警（`alert_firing`）和 kubernetes 警告事件（`event_warning`）推送到订阅的渠道。

- 只推送**新出现**的告警和事件。重启后的第一次轮询只记录现状、不发送通知，避免重新部署时重放所有已触发的告警。
- 已静默的告警不推送——静默本身就是"不要通知"。
- 重复的警告事件按事件生命周期只通知一次，不会每次出现都发。
- 仅当至少一个启用中的渠道订阅了这两类事件时，轮询才会运行。

## 渠道字段

- **名称**：渠道的唯一标识。
- **类型**：上表类型之一。
- **Endpoint**：接收端的 webhook 地址。
- **Secret**：钉钉/飞书的签名密钥；使用 `KITE_ENCRYPT_KEY` 加密存储，API 永远不会返回。
- **事件**：订阅的事件类型。
- **启用**：停用的渠道不再接收投递。

KubeOsprey 会在渠道记录上保存最近一次投递的结果（成功/失败、错误信息、时间戳），配置错误的渠道会直接暴露在设置列表里，而不是悄悄丢消息。

## 测试渠道

在设置界面点 **Test** 按钮，或直接调用接口，发送一条样例消息验证地址与签名配置：

```bash
POST /api/v1/admin/notifications/channels/:id/test
```

## API

所有接口均要求管理员权限：

| 方法     | 路径                                          | 用途           |
| -------- | --------------------------------------------- | -------------- |
| `GET`    | `/api/v1/admin/notifications/channels`        | 渠道列表       |
| `POST`   | `/api/v1/admin/notifications/channels`        | 创建渠道       |
| `PUT`    | `/api/v1/admin/notifications/channels/:id`    | 更新渠道       |
| `DELETE` | `/api/v1/admin/notifications/channels/:id`    | 删除渠道       |
| `POST`   | `/api/v1/admin/notifications/channels/:id/test` | 发送测试通知 |

这些通知所属的工作流见[变更计划](/zh/guide/change-plans)。
