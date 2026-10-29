# MCP 服务器

KubeOsprey 内置了 [Model Context Protocol](https://modelcontextprotocol.io)（MCP）服务器。Claude Code、Cursor、opencode 等 MCP 客户端可以通过它直接操作你的集群——列工作负载、读日志和事件、查指标——所有操作都在**你本人的** KubeOsprey 权限之下执行。

MCP 服务器默认关闭。

## 启用 MCP

1. 打开 `Settings`。
2. 进入 `General`。
3. 打开 `MCP Server` 开关。
4. 按需调整 `MCP Rate Limit (per minute)`（默认每用户每分钟 60 次）。
5. 保存设置。

## 认证

MCP 客户端使用 API Key 认证：

1. 打开 `Settings` → `API Keys`，创建一个 Key（谁能管理 Key 见[用户管理](../config/user-management)）。
2. 客户端在 `Authorization` 请求头中以 `kubeosprey` 前缀携带该 Key：

```
Authorization: kubeosprey<your-api-key>
```

Key 会解析为一个 KubeOsprey 用户，每次工具调用都按该用户的 RBAC 角色校验——与 Web 界面中的权限完全一致。超过速率限制的调用会被拒绝。

## 端点

服务器使用 Streamable HTTP（协议版本 2025-03-26），单一端点：

```
http(s)://<your-kubeosprey-host>/api/v1/mcp
```

### Claude Code

```json
{
  "mcpServers": {
    "kubeosprey": {
      "type": "http",
      "url": "https://kubeosprey.example.com/api/v1/mcp",
      "headers": {
        "Authorization": "kubeosprey<your-api-key>"
      }
    }
  }
}
```

### Cursor

将同样的 server 配置加入 `.cursor/mcp.json`（项目级）或全局 MCP 设置。

## 可用工具

只读工具覆盖日常巡检：

- 集群与命名空间：`list_clusters`、`list_namespaces`
- 资源：`list_resources`、`get_resource`、`list_workloads`、`get_workload`、`get_workload_history`
- Pod：`get_pod`、`get_pod_logs`
- 事件与端点：`list_events`、`list_endpoints`
- 权限：`check_permission`（SelfSubjectAccessReview）
- 指标：`get_node_metrics`、`get_pod_metrics`
- Prometheus：`prometheus_query`、`prometheus_query_range`

## 写操作走变更计划

不存在直接的写工具。Agent 想变更集群必须：

1. `create_change_plan` —— 创建一个计划
2. `append_change_plan_step` —— 追加每一步变更；每步都会先做 dry-run
3. `finalize_change_plan` —— 提交计划等待审核
4. 由人在界面中批准或拒绝
5. `apply_change_plan` —— 执行已批准的步骤

审批流、通知与回滚见[变更计划](./change-plans)。计划创建者不能审批自己的计划，执行时会再次校验权限。

## 安全说明

- 每次工具调用都按 API Key 对应用户做 RBAC 校验，并写入审计日志。
- 速率限制按用户计，可配置。
- 在设置中打开 `MCP Server` 之前，端点处于禁用状态（调用会返回错误）。
