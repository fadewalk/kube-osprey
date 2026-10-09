# Kite 吸收 Kuboard v4 能力实施计划

> 创建时间：2026-10-09
> 交底对象：下一位开发者或 AI Agent
> 参考材料：`kuboard-reference/`（Kuboard v4 行为逆向产物，全部来自黑盒探测）
> 状态：待开工。按阶段顺序执行，**Phase 1 → Phase 2 强依赖，Phase 3 独立可选**。

---

## 0. 背景与决策

Kuboard v4（闭源商业 Kubernetes 面板）经逆向分析后，有三个值得 Kite 吸收的架构设计。
本计划的目标是**在 Kite 现有 Go/Gin/GORM/React 架构内重新实现这三个能力**，对标其产品能力，不复制其任何代码、UI 或文案。

| # | Kuboard 的设计 | Kite 现状 | 吸收价值 |
|---|---|---|---|
| 1 | 内置 MCP Server（24 个工具，按用户 RBAC 裁剪） | 无 MCP，只有内嵌 AI 助手 | 让 Claude Code/Cursor/opencode 直接安全地操作集群，是差异化最大、工作量最小的一项 |
| 2 | 计划式审批（AI 只能建 plan，人批准后才 apply） | `pkg/ai/pending_session.go` 有单步确认 | 多步变更的原子审批 + dry-run + before/after 快照，安全性上一档 |
| 3 | 资源同步引擎（全量+watch 增量镜像进 SQL） | live 直连 client-go | 大集群列表/拓扑/跨集群搜索不打 API Server；代价是数据库膨胀，所以做成可选模式 |

**Clean-room 原则**：`kuboard-reference/` 里的逆向产物（OpenAPI 规格、MCP 工具契约、表结构语义）
只用于理解行为和设计**语义**。实现时：不复制 Kuboard 的代码（拿不到也不许反编译还原）、
不复制其 UI 布局/文案/图标、不复制其 API 路径命名（Kuboard 用 `api/<模块>.kuboard.cn/v4` 虚拟域名风格，Kite 保持 `/api/v1/...` 自己的风格）。

**明确不做（non-goals）**：

- 不做 Helm 应用市场（Kite 已有 Helm release 管理，市场运营是商业行为不是技术能力）
- 不做多实例部署与分布式锁（Kite 先保持单实例；Kuboard 用 `kb_instance_runtime` 心跳锁解决，等有需求再做）
- 不做许可/授权模块（Kuboard 的 license 体系是商业闭源逻辑，无吸收意义）
- 不把 PostgreSQL/MySQL 变成必选依赖（同步引擎必须兼容现有 SQLite 默认形态）

---

## 1. 技术选型决策

| 决策点 | 结论 | 理由 |
|---|---|---|
| MCP Go SDK | 优先官方 `github.com/modelcontextprotocol/go-sdk`（Streamable HTTP）；若与 Gin 集成别扭则用 `github.com/mark3labs/mcp-go` | 两者都支持 Streamable HTTP（Kuboard 用的是 Spring AI 的实现，协议版本 2025-03-26）；开工前各花半天做 spike，二选一后写进本节 |
| MCP 鉴权 | 复用现有 API Key：`Authorization: kite<apikey>` | `pkg/auth/middleware.go` 的 `RequireAPIKeyAuth` 已把 apikey 用户连同 `rbac.GetUserRoles` 结果注入 gin context，MCP 工具的 RBAC 检查零新增成本。**不引入** Kuboard 的 `Kb-Access-Key` 头 |
| 工具 RBAC | 复用 `pkg/ai/tool_authorization.go` 的 `requiredToolPermissions` + `pkg/rbac.CanAccess` | 同一套权限语义管住 Web/AI/MCP 三个入口 |
| MCP 传输 | 仅 Streamable HTTP（`POST /api/v1/mcp`，SSE 可选） | 2025-03-26 协议；Kuboard 团队踩过 SSE-only 的坑（tools/list 挂起），不要重蹈 |
| 同步引擎实现 | client-go dynamic SharedInformerFactory，自定义 Store 落 GORM | 不引 controller-runtime manager（Kite 不是 controller，manager 太重）；informer 自带 watch 重连/退避 |

---

## 2. Phase 1：内置 MCP Server

**目标**：外部 MCP 客户端（Claude Code / Cursor / opencode）用 Kite 签发的 API Key 连入，
按当前用户 RBAC 只读操作多集群。默认关闭。

### 2.1 后端

新增包 `pkg/mcpserver/`：

```
pkg/mcpserver/
  server.go        # mcp.Server 组装、工具注册、session 管理
  auth.go          # 从 gin context 取用户（RequireAuth 已完成认证），无用户则 401
  tools.go         # 工具注册表：名字、描述、inputSchema、requiredPermissions、handler
  convert.go       # 工具参数 <-> cluster.ClientSet 调用
```

路由挂在 `routes.go` 的 protected 组（`authHandler.RequireAuth()` 之后）：

```
POST /api/v1/mcp          # Streamable HTTP 单端点
DELETE /api/v1/mcp        # session 终止（协议要求）
```

**工具清单（第一版 14 个，全部只读）**——语义对照见 `kuboard-reference/mcp-tools-list.json`，
名字用 Kite 自己的命名习惯：

| Kite 工具 | 对应 Kuboard 工具 | 实现来源 |
|---|---|---|
| list_clusters | list_clusters | 新写，遍历 ClusterManager |
| list_namespaces | list_namespaces | 新写 |
| list_resources | list_k8s | 包 `pkg/resources` 现有 handler 逻辑（注意保持 cluster/namespace/`_all` 语义，AGENTS.md 红线） |
| get_resource | get_k8s | 同上 |
| list_workloads | list_workloads | 基于 `pkg/common/resource.go` Registry 过滤 Deployment/STS/DS |
| get_workload | get_workload | 新写 |
| get_workload_history | get_workload_history | 列出 owned ReplicaSet |
| get_pod / get_pod_logs / list_events | 同名 | 新写，日志 tailLines 上限 1000、超时 10s（对齐 Kuboard 的限流参数，防 DoS） |
| list_endpoints | list_endpoints | Endpoints/EndpointSlice |
| check_permission | check_permission | SelfSubjectAccessReview |
| get_node_metrics / get_pod_metrics | 同名 | metrics-server（`pkg/metrics` 已有） |
| prometheus_query / prometheus_query_range | 同名 | 包 `pkg/ai/tool_prometheus.go`，保留其中的 RBAC 限制 |

写操作工具**在本阶段不注册**——那是 Phase 2 的计划式工具，见下。

**每个工具调用的横切要求**：

1. 调用前 `requiredToolPermissions` → `rbac.CanAccess`，拒绝时返回结构化错误（工具级，不泄漏集群信息）；
2. `pkg/audit` 记录：user、tool、args 摘要、cluster、耗时、结果状态；
3. 响应带 `dangerLevel`（LOW/MEDIUM/HIGH）字段——只读工具恒为 LOW，为 Phase 2 的 UI 展示铺路。

**开关与配置**（AGENTS.md 要求三处同步）：

- `internal/config.go`：`KITE_MCP_ENABLED`（默认 false）、`KITE_MCP_RATE_LIMIT_PER_MINUTE`（默认 60）；
- `pkg/settings`：UI 可改的同名 GeneralSetting，运行时生效；
- `charts/kite`：values.yaml + deployment 模板加对应 env；
- 关闭时 `/api/v1/mcp` 返回明确的 JSON-RPC 错误（Kuboard 的做法：`code:-32001, "MCP endpoint is disabled"`）。

### 2.2 前端

`ui/src/pages/settings.tsx` 增加 MCP 区块（开关 + 限流数字），开关打开后展示连接说明卡片：
服务端 URL（当前 origin + `/api/v1/mcp`）、认证方式说明（`Authorization: kite<你的API Key>`）、
"去创建 API Key"链接跳现有 API Key 管理页。i18n 同步 `ui/src/i18n/locales/en.json` 与 `zh.json`。

### 2.3 验收标准

- `make build` + `make pre-commit` 通过；`go test ./pkg/mcpserver`、`pnpm --dir ui exec tsc --noEmit -p tsconfig.app.json` 通过
- 用真实 MCP 客户端（npx @modelcontextprotocol/inspector 或 Claude Code 配置）完成 initialize → tools/list → 任选 3 个工具调用，RBAC 拒绝路径与审计记录可查
- 关闭开关后端点返回 disabled 错误；无 API Key 返回 401
- 审计表里每次 MCP 调用可追溯到 user + tool

---

## 3. Phase 2：计划式审批（Change Plan）

**目标**：MCP/AI 的**写操作**只能以"计划"形式提交，具备相应权限的人在 UI 审批后才真正执行。
这是 Kuboard 最值得学的设计：它没有任何直接 apply 工具，AI 无法绕过人工闸门。

### 3.1 数据模型（`pkg/model/` 新增两个 GORM 模型）

语义对照 Kuboard 的 `kb_mcp_change_plan` / `kb_mcp_change_plan_step`（见
`kuboard-reference/REVERSE-ENGINEERING.md` 第四节），列名用 Kite 自己的约定：

```
change_plans:
  id, user_id, agent_session_id, client_name, description
  status        # 状态机见 3.2
  danger_level  # steps 中最高等级
  approval_token      # crypto/rand 32B，一次性，审批时校验并置空
  approver_user_id, approved_at, expires_at（默认 30 分钟）
  created_at, updated_at

change_plan_steps:
  id, plan_id, seq, tool_name, description
  cluster, namespace, resource          # 目标对象描述
  danger_level, status
  args_json, before_json, after_json    # 快照
  dry_run_supported, dry_run_status, dry_run_response
  apply_status, apply_response, error_message
  applied_at
```

### 3.2 状态机与工具

```
plan:  CREATED → PENDING → (APPROVED → APPLYING → APPLIED | FAILED) | REJECTED | EXPIRED
step:  PENDING → DRY_RUN_OK | DRY_RUN_FAILED | DRY_RUN_UNSUPPORTED → APPLIED | FAILED | SKIPPED
```

注册 5 个 MCP 工具（Phase 1 的 `tools.go` 追加）：

| 工具 | 语义（对照 Kuboard） |
|---|---|
| create_change_plan | 创建空 plan（CREATED，UI 不可见不可审批） |
| append_change_plan_step | 追加一步：目标 cluster/namespace/resource + 想写入的对象（after_json）；追加时**立即做 server-side dry-run**（`dryRun: All`），不支持 dry-run 的资源标记 `DRY_RUN_UNSUPPORTED` |
| finalize_change_plan | CREATED 原子转 PENDING（单事务，防并发追加），此时 UI 审批中心可见 |
| get_change_plan_status | AI 轮询审批结果与每步执行状态 |
| apply_change_plan | 仅当状态 APPROVED 且 approval_token 匹配时按 seq 顺序执行；任一步失败即停止，剩余标 SKIPPED，plan 落 FAILED |

**红线**：

- `apply_change_plan` 之外的任何工具/路径都不得触发真实写操作；
- approval_token 一次性消费（审批动作在事务里校验+置空），防止重放；
- 审批人自己不能是 plan 创建者之外还兼任绕过：**创建者 != 审批者**强制校验（Kuboard 未做，我们做得更好）；
- 每步执行前再验一次 RBAC（token 从创建到审批之间用户权限可能被回收——Kuboard 只在建 plan 时验，这是它的缺口）。

### 3.3 前端：审批中心

- 新页面 `ui/src/pages/change-plans.tsx`：PENDING 列表（badge 显示 danger_level 颜色）、
  详情抽屉里用 **Monaco DiffEditor**（`@monaco-editor/react` 已在依赖里）展示每步 before/after、dry-run 结果；
- 批准/拒绝按钮走 `POST /api/v1/change-plans/:id/approve|reject`（`RequireAuth` + rbac：`resource=change_plan, verb=approve`，只有 admin 或被授权角色可见）；
- 导航入口带待审数量 badge（现有导航模式上加）。

### 3.4 验收标准

- 状态机表驱动测试：全路径（含 EXPIRED、token 重放、并发 finalize、审批人=创建人拒绝、中途权限回收）
- 集成测试：MCP 客户端走完 create → append(dry-run) → finalize →（UI/HTTP 审批）→ apply，真实集群上创建再回滚一个 ConfigMap
- 审计完整：审批动作、每步 apply 结果都有记录
- Phase 1 的内嵌 AI 助手写路径也切到同一套 plan 机制（`pkg/ai/pending_session.go` 的单步确认保留为快速通道，多步必须走 plan）

---

## 4. Phase 3：资源同步引擎（可选模式，默认关闭）

**目标**：把集群核心资源镜像进 SQLite/PG，列表与全局搜索优先走库，直连作为 fallback。
解决两个真实痛点：大集群列表慢、跨集群全局搜索现在做不到。

**默认不同步全部资源类型**——只同步白名单（起步：pods, deployments, statefulsets, daemonsets,
services, ingresses, configmaps, secrets(仅元数据不落 data 字段), nodes, namespaces, events(仅近期)，
在 settings 里可增删）。Kuboard 全量同步 + MariaDB 是它的架构选择，Kite 的默认形态是单二进制
+SQLite，**膨胀控制是本阶段第一约束**。

### 4.1 实现

新增 `pkg/syncengine/`：

- 每 (cluster, gvr) 一个 informer（dynamic client + SharedInformerFactory），事件去抖后 upsert：
  主键 `(uid, cluster)`，更新条件 `resourceVersion` 新于已存值；
- 表 `sync_objects`：语义对照 Kuboard `kb_cluster_sync_data`（保留 `owner_uid` 三级链做拓扑、
  `name searchable` 列），object JSON 原文存储；
- 表 `sync_tasks`：每 (cluster, gvr) 的同步状态/水位/错误（对照 `kb_cluster_sync_task`）；
- 删除事件靠 informer DeleteFunc 软删（`deleted_at`），保留窗口后物理清理；
- 停机重启用 List 重建全量（informer 本来就 ListAndWatch，无需自己管水位恢复）。

### 4.2 消费路径切换

- `pkg/resources` 列表接口与 `pkg/search` 全局搜索：模式开启时读 sync 表（SQL 过滤 + 分页），
  读不到（类型不在白名单）自动 fallback 现有直连路径——**对前端完全透明**；
- 拓扑视图（owner 链查询）只在同步模式下提供，作为新能力上线。

### 4.3 验收标准

- 对 500+ pod 的测试集群：同步稳态内存增量 < 200MB，SQLite < 500MB；列表 P95 比直连路径快
- 断线重连、API Server 滚动重启后 informer 自愈，无全量重刷（验证 resync 不打爆 API Server）
- 关闭开关后一切回到现状，sync 表数据留存策略可配（默认 7 天物理清理）

---

## 5. 里程碑与顺序

| 阶段 | 交付物 | 预估 | 依赖 |
|---|---|---|---|
| P1 | MCP Server（14 只读工具 + 开关 + 审计 + 设置页） | 1~2 周 | 无 |
| P2 | Change Plan + 审批中心 + AI 写路径接入 | 2 周 | P1 |
| P3 | 同步引擎（白名单模式）+ 搜索/列表切换 + 拓扑 | 2~3 周 | 无（可并行，建议最后做） |

每阶段独立可发布、默认关闭、可独立回滚（P3 回滚 = 关开关）。

## 6. 测试与工程要求（用户明确要求，覆盖 AGENTS.md 默认不写测试的约定）

- 本计划涉及的**状态机、token 消费、RBAC 拒绝路径、dry-run 语义**必须有表驱动测试；
- MCP 协议层至少一条端到端集成测试（真实 HTTP 握手 → tools/list → tools/call）；
- 每阶段完成跑 `make pre-commit` 全量；涉 chart 改动跑 `helm lint charts/kite` + `scripts/validate-chart.sh`；
- 真实集群验证用 Colima（`kubectl config use-context colima`，与 HANDOFF.md 第 8 条一致）。

## 7. 风险清单

| 风险 | 缓解 |
|---|---|
| MCP Go SDK 与 Gin 集成别扭 | P1 第一天做 spike，两个候选各跑通 initialize 再定 |
| SQLite 写放大（P3） | 去抖 + 批量事务 upsert + 白名单默认小；必要时引导切 PG |
| secrets 同步泄密 | 白名单默认排除 secrets；用户显式勾选也只存元数据不存 data |
| 工具误操作 | P2 审批闸门 + 每步 dry-run + 审批后二次 RBAC 复查 |
| 与 Kuboard 的法律边界 | 只做语义对标（本计划反复强调），无代码/UI/文案/路径复制 |
