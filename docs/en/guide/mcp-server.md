# MCP Server

KubeOsprey ships with a built-in [Model Context Protocol](https://modelcontextprotocol.io) server. It lets MCP clients such as Claude Code, Cursor, or opencode operate your clusters directly — listing workloads, reading logs and events, querying metrics — always under **your own** KubeOsprey permissions.

The MCP server is off by default.

## Enable MCP

1. Open `Settings`.
2. Go to `General`.
3. Turn on `MCP Server`.
4. Adjust `MCP Rate Limit (per minute)` if needed (default 60 calls per user).
5. Save the settings.

## Authentication

MCP clients authenticate with an API key:

1. Open `Settings` → `API Keys` and create a key (see [User Management](../config/user-management) for who can manage keys).
2. Clients send the key in the `Authorization` header using the `kubeosprey` scheme:

```
Authorization: kubeosprey<your-api-key>
```

The key resolves to a KubeOsprey user, and every tool call is checked against that user's RBAC roles — the same permissions that apply in the web UI. Calls that exceed the rate limit are rejected.

## Endpoint

The server speaks Streamable HTTP (protocol version 2025-03-26) at a single endpoint:

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

Add the same server block to `.cursor/mcp.json` (project) or the global MCP settings.

## Available Tools

Read-only tools cover day-to-day inspection:

- Clusters and namespaces: `list_clusters`, `list_namespaces`
- Resources: `list_resources`, `get_resource`, `list_workloads`, `get_workload`, `get_workload_history`
- Pods: `get_pod`, `get_pod_logs`
- Events and endpoints: `list_events`, `list_endpoints`
- Permissions: `check_permission` (SelfSubjectAccessReview)
- Metrics: `get_node_metrics`, `get_pod_metrics`
- Prometheus: `prometheus_query`, `prometheus_query_range`

## Write Operations Go Through Change Plans

There is no direct write tool. An agent that wants to change the cluster must:

1. `create_change_plan` — open a plan
2. `append_change_plan_step` — add each intended change; every step is dry-run first
3. `finalize_change_plan` — submit the plan for review
4. A human approves or rejects it in the UI
5. `apply_change_plan` — execute the approved steps

See [Change Plans](./change-plans) for the approval workflow, notifications, and rollback. The plan creator cannot approve their own plan, and permissions are re-checked at apply time.

## Security Notes

- Every tool call is RBAC-checked against the API key's user and written to the audit log.
- Rate limiting is per user and configurable.
- The endpoint is disabled (and returns an error) until `MCP Server` is turned on in settings.
