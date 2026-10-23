# Notification Channels

Notification Channels let KubeOsprey push Change Plan lifecycle events to your team's chat tools. Administrators configure channels in **Settings → Notification Channels**; each channel subscribes to the events it cares about.

![Notification channels settings page](/screenshots/notification-channels.png)

## Supported channel types

| Type       | Protocol                                                                        | Notes                                        |
| ---------- | ------------------------------------------------------------------------------- | -------------------------------------------- |
| `dingtalk` | DingTalk group robot, Markdown payload                                          | Optional HMAC-SHA256 signing via secret      |
| `wecom`    | WeCom (企业微信) group bot, Markdown payload                                     | Messages longer than 4096 bytes are truncated |
| `feishu`   | Feishu (Lark) bot, text payload                                                 | Optional timestamp-based signing via secret   |
| `slack`    | Slack incoming webhook, `{"text": ...}`                                          |                                              |
| `webhook`  | Generic receiver, JSON `{"title": ..., "text": ...}`                            | Point it at any HTTP endpoint                 |

## Events

| Event          | Meaning                                |
| -------------- | -------------------------------------- |
| `plan_pending` | A change plan is awaiting approval      |
| `plan_applied` | A change plan finished successfully     |
| `plan_failed`  | A change plan finished with failures    |

Each channel stores its subscribed events as a comma-separated list, so one channel can receive several event kinds.

## Channel fields

- **Name** — unique identifier of the channel.
- **Type** — one of the types above.
- **Endpoint** — the webhook URL of the receiver.
- **Secret** — signing secret for DingTalk/Feishu; stored encrypted with `KITE_ENCRYPT_KEY` and never returned by the API.
- **Events** — subscribed event kinds.
- **Enabled** — disabled channels stop receiving deliveries.

KubeOsprey records the delivery result of the last dispatch per channel (success/failure, error message, timestamp), so a misconfigured channel is visible in the settings list instead of silently dropping messages.

## Testing a channel

Use the **Test** button in the settings UI, or call the API directly, to send a sample message and verify the endpoint and signing configuration:

```bash
POST /api/v1/admin/notifications/channels/:id/test
```

## API

All endpoints require administrator access:

| Method   | Path                                          | Purpose                    |
| -------- | --------------------------------------------- | -------------------------- |
| `GET`    | `/api/v1/admin/notifications/channels`        | List channels              |
| `POST`   | `/api/v1/admin/notifications/channels`        | Create a channel           |
| `PUT`    | `/api/v1/admin/notifications/channels/:id`    | Update a channel           |
| `DELETE` | `/api/v1/admin/notifications/channels/:id`    | Delete a channel           |
| `POST`   | `/api/v1/admin/notifications/channels/:id/test` | Send a test notification |

See [Change Plans](/guide/change-plans) for the workflow these notifications belong to.
