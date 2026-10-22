# Change Plans

Change Plans add an approval gate in front of sensitive operations. Instead of applying a change immediately, KubeOsprey records it as a plan with explicit steps; an approver reviews and approves it; only then is the change executed, and every applied plan can be rolled back.

![Change Plans](/screenshots/audit.webp)

## When plans are created

Two switches in **Settings → General** control which operations are gated:

- **AI change plan mode (`aiChangePlanMode`)** — when enabled, write operations requested through the [AI Assistant](./ai-assistant) do not run directly. The assistant records the intended steps as a plan and hands it to a human for approval.
- **Webhook change plan mode (`webhookChangePlanMode`)** — when enabled, Helm upgrades submitted through the CI webhook endpoint (`POST /api/v1/webhooks/helm-upgrade`) are held as pending plans instead of executing immediately. This turns your CI/CD pipeline into a review-before-deploy flow.

## Plan lifecycle

```
created → pending → approved → applying → applied
                  ↘ rejected        ↘ failed
pending → expired (approval not given before the deadline)
```

- A plan contains one or more **steps**, each describing one operation together with its dry-run result, so the approver sees what would happen before approving.
- **Approval requires a separate permission** — the person who created the plan cannot approve it themselves.
- Pending plans expire if no decision is made in time.
- Applied plans support **rollback** (see [Resource History](./resource-history) for the underlying revision trail).

## Notifications

KubeOsprey can push plan events (`plan_pending`, `plan_applied`, `plan_failed`) to DingTalk, WeCom, Feishu, Slack, or a generic webhook. See [Notification Channels](../config/notification-channels).

## API

| Method | Path                                    | Purpose                          |
| ------ | --------------------------------------- | -------------------------------- |
| `GET`  | `/api/v1/change-plans`                  | List plans                       |
| `GET`  | `/api/v1/change-plans/:id`              | Plan detail with steps           |
| `GET`  | `/api/v1/change-plans/pending-count`    | Badge count of pending plans     |
| `POST` | `/api/v1/change-plans/:id/approve`      | Approve a pending plan           |
| `POST` | `/api/v1/change-plans/:id/reject`       | Reject a pending plan            |
| `POST` | `/api/v1/change-plans/:id/rollback`     | Roll back an applied plan        |
