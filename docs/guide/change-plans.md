# 变更计划

变更计划（Change Plan）为敏感操作增加审批闸门：变更不再立即执行，而是先记录为带明确步骤的计划，由审批人审核通过后才执行，且每个已执行的计划都支持回滚。

![变更计划](/screenshots/change-plans.png)

## 计划何时产生

**Settings → General** 中的两个开关控制哪些操作需要走计划：

- **AI 变更计划模式（`aiChangePlanMode`）**：启用后，通过 [AI 助手](./ai-assistant)发起的写操作不再直接执行。助手把意图操作记录为计划，交由人工审批。
- **Webhook 变更计划模式（`webhookChangePlanMode`）**：启用后，通过 CI webhook 接口（`POST /api/v1/webhooks/helm-upgrade`）提交的 Helm 升级会挂起为待审批计划而非立即执行，把 CI/CD 流水线变成"先审批、后部署"的流程。
- Webhook 负载支持可选的 `chartVersion`：升级时从 Release 所属仓库目录中拉取该版本，而不是沿用当前 Chart。若同时提供 `chartUrl` 则以 `chartUrl` 为准；两者都缺省时，使用当前已部署的 Chart 配合提交的 values 升级。

## 计划生命周期

```
created → pending → approved → applying → applied
                  ↘ rejected        ↘ failed
pending → expired（超时未审批）
```

- 一个计划包含一个或多个**步骤**，每个步骤描述一次操作及其 dry-run 结果，审批人批准前就能看到会发生什么。
- **审批需要独立权限**——计划的创建者不能审批自己的计划。
- 待审批计划超时未处理会自动过期。
- 已执行的计划支持**回滚**（底层修订记录见[资源历史](./resource-history)）。

## 通知

KubeOsprey 可以把计划事件（`plan_pending`、`plan_applied`、`plan_failed`）推送到钉钉、企业微信、飞书、Slack 或通用 webhook。参见[通知渠道](../config/notification-channels)。

## API

| 方法   | 路径                                    | 用途                     |
| ------ | --------------------------------------- | ------------------------ |
| `GET`  | `/api/v1/change-plans`                  | 计划列表                 |
| `GET`  | `/api/v1/change-plans/:id`              | 计划详情（含步骤）       |
| `GET`  | `/api/v1/change-plans/pending-count`    | 待审批数量（角标）       |
| `POST` | `/api/v1/change-plans/:id/approve`      | 审批通过                 |
| `POST` | `/api/v1/change-plans/:id/reject`       | 拒绝                     |
| `POST` | `/api/v1/change-plans/:id/rollback`     | 回滚已执行的计划         |
