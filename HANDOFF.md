# Kite 项目交接文档

> 更新时间：2026-10-10
> 当前分支：`feat/kubepi-value-mvp`
> 当前 HEAD：`dfd2905 feat: add workload quick operations`（**未推送**，领先 origin/main 12 个提交，fadewalk/kite 上游无此分支更新）

本文档用于把当前 Kite / KubePi 能力吸收工作交接给下一位开发者或 AI Agent。除非用户明确要求，不要重做已经完成的工作，也不要把本地工具目录加入提交。

## 1. 项目定位和总体决策

Kite 的目标不是复制 KubePi，而是建设：

> 安全、可靠、支持 Cluster Agent 和 AI 辅助运维的多集群 Kubernetes 控制面。

KubePi 只作为产品能力和交互体验参考。不要复制 KubePi 的 Go/Vue/Angular/gotty 源码；KubePi 使用 GPLv3 及额外条款，当前实现坚持在 Kite 的 Gin/GORM/React/TypeScript 架构中重新实现。

当前明确的产品边界：

- Harbor/Nexus/Docker Registry 由外部镜像平台负责，Kite 只做可选的只读镜像引用、Tag/digest 查询和 imagePullSecret 辅助，不重做镜像仓库后台。
- 原始 PromQL 被视为集群级可观测性能力，不做脆弱的 namespace 字符串改写；多租户隔离应由 Prometheus/Thanos/Mimir 等指标架构或受限查询模型提供。
- Cluster Agent、kubectl Terminal、Node Terminal 属于高权限能力，优先做最小权限、Break-glass、Token 生命周期和审计。

## 2. 已完成提交

### 原 KubePi Value MVP

| Commit | 内容 |
| --- | --- |
| `dc7f229` | Pod 文件路径校验、上传文件名校验、文件列表错误语义修复 |
| `73247e1` | Helm Chart `.tgz` 导出 API 和详情页导出按钮 |
| `2d6d49c` | PromQL instant/range query 后端 API |
| `fb6eb22` | PromQL Explorer React 页面、路由、侧边栏和 i18n |

### 第一轮生产安全基线

| Commit | 内容 |
| --- | --- |
| `d7f2f3c` | 中英文生产安全基线文档和 VitePress 导航 |
| `a8f8a73` | 启动密钥预检、数据库初始化 fail-closed、`/readyz`、本地开发兼容开关 |
| `16f3718` | Helm 加密 Secret 随机生成/保留、DB 类型和 DSN 校验、SQLite `Recreate`、readiness 使用 `/readyz` |
| `eeda143` | WebSocket 默认同源校验，支持 `CORS_ALLOWED_ORIGINS` 显式 allowlist |
| `c2b3347` | 通用资源、CRD、Apply、AI、Helm 历史记录统一敏感字段脱敏 |
| `cbf11f7` | 新建 GeneralSetting 中 kubectl Terminal 默认关闭，前端回退值同步 |

### Docker、权限和 Break-glass

| Commit | 内容 |
| --- | --- |
| `5edd33b` | `.dockerignore`、固定 Dockerfile pnpm 版本 `10.33.0`，缩小构建上下文并提高可复现性 |
| `ecd8351` | Cluster Agent 权限档位 `readOnly` / `operator` / `admin` / `custom` |
| `b29d825` | Cluster Agent 权限档位 Manifest 单测和 API/UI 字段修正 |
| `ae49fc7` | kubectl Terminal 改为每会话临时 ServiceAccount/ClusterRoleBinding/Pod 并清理 |
| `4ca0290` | Break-glass 清理注释和格式收尾 |
| `55777e9` | 移除 `pkg/model` 未使用的 `initErr` 变量（golangci-lint 报错） |
| `1094770` | Cluster Dialog 仅格式化收尾 |
| `b7884d4` | kubectl Terminal 权限模式可配置：`kubectlPermissionMode`（`admin`/`operator`/`custom`，默认 `operator`）+ `kubectlClusterRole`；`operator` 模式每会话创建受限 ClusterRole（不含 Secrets/RBAC/节点写），`custom` 绑定已有 ClusterRole；会话结束清理临时 ClusterRole |
| `84b4d38` | 收紧 Cluster Agent operator 档位：补齐 pods/exec、attach、portforward，events/namespaces/nodes/PV/StorageClass/CRD/metrics 只读，Secrets 只读（Kite 控制台与 Helm Release 依赖），不授 RBAC/通配符；文档补充 roleRef 不可变、切换档位须先删 Binding |
| `94aa104` | Node Terminal 独立 break-glass：`nodeTerminalEnabled` 默认关闭，会话要求管理员角色 + 显式开启；bootstrap capabilities 增加 `nodeTerminalEnabled`，前端节点详情页按管理员+开关隐藏 Terminal tab；设置页增加独立开关和警告文案 |

## 3. 当前关键实现位置

### 启动、安全和健康检查

- `pkg/common/common.go`
  - `ValidateEncryptionKey`
  - `KITE_ALLOW_INSECURE_DEFAULTS`
  - WebSocket Origin 校验
- `app.go`
  - `initializeApp`
  - `appReady`
  - 数据库/GeneralSetting 失败时拒绝启动
- `routes.go`
  - `/healthz`：进程存活，保持兼容
  - `/readyz`：应用初始化完成后才返回 200
- `pkg/model/model.go`
  - `InitDB() error`
- `Makefile`
  - `make dev` 仅本地开发设置 `KITE_ALLOW_INSECURE_DEFAULTS=true`

生产环境不要设置 `KITE_ALLOW_INSECURE_DEFAULTS=true`。

### Helm 和 Docker

- `Dockerfile`
  - 固定安装 `pnpm@10.33.0`
- `.dockerignore`
  - 排除 `.codegraph`、`.zcode`、依赖目录、静态生成物、本地数据库等
- `charts/kite/templates/secret.yaml`
  - 新 Secret 随机生成 `KITE_ENCRYPT_KEY`
  - `lookup` 保留已有 Secret 的加密密钥和 JWT
  - 非法 `db.type` / 缺少非 SQLite DSN 时模板失败
- `charts/kite/values.yaml`
  - `encryptKey` 默认空
  - SQLite 默认 `deploymentStrategy: Recreate`
  - readiness 默认 `/readyz`
- `scripts/validate-chart.sh`
  - 已修正 PostgreSQL 测试使用 `db.dsn`
- `docs/config/security-hardening.md`
- `docs/zh/config/security-hardening.md`

### Cluster Agent 权限档位

- `pkg/model/cluster.go`
  - `ClusterAgentPermission`
  - `ClusterAgentRole`
  - 常量：`readOnly`、`operator`、`admin`、`custom`
- `pkg/cluster/cluster_handler.go`
  - 创建 Cluster Agent 时校验权限档位
  - 新建 Agent 默认 `readOnly`
  - 旧数据库空值兼容为 `admin`
  - Manifest endpoint 根据 Token 读取权限档位
- `pkg/clusteragent/manager.go`
  - `RegistrationPermission`
- `pkg/clusteragent/manifest.go`
  - `GenerateManifestWithPermission`
  - `readOnly`：只读 ClusterRole
  - `operator`：限定常见工作负载、网络和批处理资源的读写
  - `admin`：兼容旧行为，绑定 `cluster-admin`
  - `custom`：绑定已有 ClusterRole，不创建同名 Role
- `ui/src/components/settings/cluster-dialog.tsx`
  - Cluster Agent 创建权限选择
- `pkg/clusteragent/manifest_test.go`
  - 权限档位 Manifest 覆盖

注意：operator 档位已于 2026-09-10 在 Colima k3s 用 `kubectl auth can-i --list` 实测验证（`84b4d38`）：工作负载读写 + 集群只读 + Secrets 只读，Secret/RBAC 写、节点写均确认拒绝。已知运维注意事项：Kubernetes 的 ClusterRoleBinding `roleRef` 不可变，切换档位必须先删除 `kite-cluster-agent` Binding 再应用新 Manifest（已写入安全基线文档中英文版）。后续如继续收紧，按实际页面 403 反馈补充资源，不要直接恢复 `cluster-admin`。

### Node Terminal Break-glass（基础门控已完成）

- `pkg/terminal/node_terminal_handler.go`
  - 会话要求：nodes exec RBAC + 管理员角色 + `nodeTerminalEnabled` 显式开启（默认关）
- `pkg/auth/bootstrap_handler.go`
  - capabilities 暴露 `nodeTerminalEnabled`
- `ui/src/pages/node-detail.tsx`
  - 非管理员或未开启时隐藏 Terminal tab
- `ui/src/components/settings/general-management.tsx`
  - 独立开关 + 特权 shell 警告文案（中英文）

Pod 本身仍是 `HostNetwork/HostPID/HostIPC + HostPath / + privileged`。剩余待做：节点范围限制、会话超时、审计日志，以及评估更小权限替代方案（见 P0-C 剩余项）。

### kubectl Terminal Break-glass（已完成权限模式可配置）

- `pkg/terminal/kubectl_terminal_handler.go`
  - 每次会话创建临时 ServiceAccount
  - `kubectlPermissionMode` 三档：
    - `admin`：绑定 `cluster-admin`，必须显式选择；
    - `operator`（默认）：每会话创建临时受限 ClusterRole `{session}-role`（工作负载读写 + 集群只读 + metrics 只读；不含 Secrets、RBAC、节点写）；
    - `custom`：绑定 `kubectlClusterRole` 指定的已有 ClusterRole（设置页强制非空）；
  - 会话结束或失败时清理 Pod、Binding、临时 ClusterRole、ServiceAccount
- `pkg/model/general_setting.go`
  - `KubectlPermissionMode` / `KubectlClusterRole` 字段与归一化（空值/非法值回退 `operator`）
- `pkg/settings/handler.go`
  - 设置 API 暴露/校验两个新字段
- `ui/src/components/settings/general-management.tsx`
  - 权限模式下拉、custom ClusterRole 输入、保存校验、i18n
- `pkg/terminal/terminal_handler_test.go`
  - 覆盖 operator/admin/custom 三档绑定、operator 角色不含 secrets、清理不删已有 custom ClusterRole

历史问题（旧 HEAD）"临时会话固定绑定 `cluster-admin`" 已由 `b7884d4` 解决。

### 历史脱敏

- `pkg/model/resource_history.go`
  - `RedactResourceYAML`
  - 脱敏 `password`、`token`、`secret`、`privateKey`、`clientSecret`、`kubeconfig`、`apiKey` 等字段
  - Kubernetes Secret 的 `data` 和 `stringData` 整体替换为 `[REDACTED]`
- 已接入：
  - `pkg/resources/generic_resource_handler.go`
  - `pkg/resources/cr_handler.go`
  - `pkg/resources/apply_handler.go`
  - `pkg/ai/tool_resource_execution.go`
  - `pkg/helmutil/history.go`
- 测试：`pkg/model/custom_type_test.go`

## 4. 本地验证环境和证据

### Colima

当前本机保留 Colima Kubernetes：

- Colima：`0.10.3`
- context：`colima`
- runtime：Docker
- Kubernetes：k3s `v1.35.0+k3s1`
- 节点：`colima`，状态 `Ready`
- Helm：`v4.2.4`
- kubectl：`v1.37.0`

查看状态：

```bash
colima status
kubectl config current-context
kubectl get nodes -o wide
helm list -A
```

### Helm 实际验证

已成功执行：

```bash
helm lint charts/kite
bash scripts/validate-chart.sh
```

已在 Colima 中完成：

- Helm 安装；
- SQLite + PVC；
- 用户创建和登录；
- Helm upgrade；
- 删除并重建 Pod；
- 数据和加密字段恢复；
- Helm rollback；
- `/healthz`、`/readyz`、`/api/v1/version`。

测试 namespace `kite-test` 和 `kite-persist` 已清理，Colima 集群保留。

### 当前已通过的代码检查

```bash
go test ./...
go vet ./...
pnpm --dir ui exec tsc --noEmit -p tsconfig.app.json
pnpm --dir docs exec vitepress build .
git diff --check
```

### Docker 构建

正式 Dockerfile 已成功构建：

```bash
docker build -t kite:dockerfile-test .
```

Linux arm64 临时构建也已成功，用于 Colima k3s：

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o /tmp/kite/kite .
```

## 5. 当前工作区状态

源码分支干净。以下本地目录已加入 `.gitignore`，**不要提交**：

```text
.codegraph/  .zcode/  /kuboard-reference/  /architecture-diagrams/  .playwright-mcp/
```

`kuboard-reference/` 是 Kuboard 逆向参考材料（clean-room 红线：只读对标语义，
不进仓库、不复制代码/UI/文案/API 路径）。本地开发库 `dev.db` 的 admin 密码
在 2026-10-10 被重置为 `KiteDev#2026`（bcrypt 直接写库，用户可在 UI 改回）。
另外注意：Kite 的角色变更会使已发 JWT 立即失效（上游行为），改完 RBAC 后需重新登录。

## 6. 下一阶段优先级

> 2026-10-10 更新：第 9 节的 Kuboard 三阶段与缺口分析 G1/G2/G3/G4/G9 均已交付
>（见第 10 节）。剩余为 G5/G6/G7/G8/G10 低优先级项与长期多集群观察，
> 详见 `KUBOARD-GAP-ANALYSIS.md`。

### P0：先完成高权限治理 ✅ A/B/C 基础项均已完成

#### A. 将 kubectl Break-glass 的 `cluster-admin` 改为可配置 ✅ 已完成（`b7884d4`）

已完成内容：

- GeneralSetting 新增 `kubectlPermissionMode`（`admin` / `operator` / `custom`，默认 `operator`）和 `kubectlClusterRole`；
- `admin` 必须显式选择，兼容旧 `cluster-admin` 行为；
- `operator` 每会话创建受限 ClusterRole，绑定临时 SA；
- `custom` 绑定已有 ClusterRole，设置保存时强制非空；
- kubectl Terminal 仍默认关闭；每会话仍创建临时 SA/Binding；会话结束清理全部临时对象（含临时 ClusterRole）。

#### B. 收紧 Cluster Agent operator 规则 ✅ 已完成（`84b4d38`）

- `operator` 档位已按 Kite 功能面重写并在 Colima k3s 用 `kubectl auth can-i --list` 实测：工作负载读写（pods 含 exec/attach/portforward、deployments/statefulsets/daemonsets/replicasets、jobs/cronjobs、svc/ep/configmap/pvc、ingress/netpol、PDB/HPA），只读（events/namespaces/nodes/PV/StorageClass/CRD/metrics/Secrets）；
- 明确排除：Secret 写、RBAC、节点写、通配符；
- 已在文档（中英文）记录 roleRef 不可变、切换档位先删 Binding 的注意事项。

后续如继续收紧：按实际页面 403 反馈补充资源，不要直接恢复 `cluster-admin`。

#### C. Node Terminal 单独 Break-glass ✅ 基础门控已完成（`94aa104`）

已完成：

- 独立 `nodeTerminalEnabled`，默认关闭；
- 会话要求 nodes exec RBAC + 管理员角色 + 显式开启；
- bootstrap capabilities + 前端 tab 隐藏 + 设置页开关与警告。

仍待做（后续批次）：

- 节点范围限制、会话超时、审计日志；
- 评估是否能用更小权限替代 privileged/root host shell（Pod 仍为 `HostNetwork/HostPID/HostIPC + HostPath / + privileged`）。

### P1：可靠性和供应链

1. Cluster Agent Token 轮换、吊销和过期；
2. Cluster Agent Manifest 镜像固定 digest；
3. kubectl/node/Agent 镜像固定版本，不使用 `latest`；
4. Passkey、Cluster Agent session 和 scheduler 的多副本共享存储；
5. 数据库备份、恢复和加密密钥轮换流程；
6. 将 `charts/**`、`deploy/**`、`scripts/**` 纳入 CI 路径触发；
7. CI 自动执行 Helm lint/template 和安全权限断言；
8. 增加 SBOM、镜像扫描和依赖漏洞检查。

### P1：已有能力产品化

1. PromQL Explorer 的 Metric/Label/Series 查询；
2. PromQL 图表、查询历史和模板；
3. 增加独立 `metrics.query` 权限；
4. Helm Release 与 Chart diff；
5. Helm Values 和 Rendered Manifest 导出；
6. Helm Auto Upgrade 的失败重试、变更审计和回滚预览；
7. Pod 文件大小限制、符号链接处理和操作审计。

### P2：选择性吸收 KubePi

只在明确产品需求下做：

- Grafana 只读 Dashboard 集成；
- SAML 2.0 企业身份兼容；
- 轻量 Harbor 镜像 Tag/digest 和 imagePullSecret 集成；
- Kubernetes 原生 RBAC 权限预览/委派。

不要做：

- Harbor 完整管理后台；
- KubePi Vue/Angular/gotty 迁移；
- AppMarket CRD；
- 原始 PromQL 的脆弱 namespace 字符串改写；
- 直接复制 KubePi GPL 源码。

## 7. 下一位 Agent 的启动步骤

1. 阅读本文件和 `AGENTS.md`。
2. 确认分支：

```bash
git status --short --branch
git log -8 --oneline --decorate
```

3. 保留 `.codegraph/`、`.zcode/` 为本地未跟踪目录。
4. 修改前先用 CodeGraph 或 Read 检查当前实现，不要假设历史文件仍未变化。
5. 每个安全改动独立提交，建议提交格式：

```text
fix: make kubectl terminal permission mode configurable
fix: tighten cluster agent operator permissions
fix: add node terminal break-glass controls
```

6. 每批至少运行：

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
pnpm --dir ui exec tsc --noEmit -p tsconfig.app.json
git diff --check
```

7. 涉及 Chart 时补跑：

```bash
helm lint charts/kite
bash scripts/validate-chart.sh
helm template kite charts/kite --namespace kite-system
```

8. 涉及真实集群权限时使用 Colima：

```bash
kubectl config use-context colima
kubectl get nodes
kubectl auth can-i --list --as=system:serviceaccount:<namespace>:<service-account>
```

9. 不修改或提交：

```text
static/
kite
bin/
node_modules/
.codegraph/
.zcode/
```

---

## 9. 下一阶段：吸收 Kuboard v4 能力（2026-10-09 新增）

完整实施计划见仓库根目录 **`KUBOARD-PARITY-PLAN.md`**，参考材料（Kuboard v4 行为逆向产物：
逆向报告、完整 OpenAPI 规格、MCP 工具契约）在 **`kuboard-reference/`**。

- Phase 1：内置 MCP Server（默认关，复用 API Key + RBAC + 审计）
- Phase 2：计划式审批（AI/MCP 写操作必须人审批，含 dry-run 与 before/after 快照）
- Phase 3：资源同步引擎（白名单模式，默认关）

注意 clean-room 红线：只对标语义，不复制 Kuboard 的代码、UI、文案或 API 路径命名。
开工前先读计划的"0. 背景与决策"和"1. 技术选型决策"两节。


## 10. Kuboard 吸收交付（2026-10-10 本轮）

上游 main（v0.16.0，含插件系统 #728）已合并进本分支（`e42ac1c`，冲突解法见提交）。
Kuboard 三阶段全部交付并真实集群（colima-xwssd k3s v1.35）验收通过。合并时工作区
曾出现文件损坏（NUL 字节 + 无关项目代码拼接），确认非有效工作后已 stash 并 drop。

### 提交清单

| Commit | 内容 |
| --- | --- |
| `e42ac1c` | 合并 origin/main（v0.16.0 插件系统等 15 提交），解决 4 处冲突 |
| `ac6282f` | 修复合并引入的 `/prometheus/query` 路由重复注册（gin 启动 panic），保留上游 QueryPrometheus 对（插件 SDK 契约） |
| `e7ccef3` | **P1 内置 MCP Server**：`pkg/mcpserver`（官方 go-sdk v1.3 Streamable HTTP，`POST/GET/DELETE /api/v1/mcp`），16 个只读工具，复用 API Key + RBAC（`ai.RunTool` 限定只读白名单），审计落 `mcp_audit_logs`，限流默认 60 次/分/用户，默认关闭 |
| `bf5b4c8` | **P2 计划式审批**：`pkg/changeplan` + 5 个 MCP 计划工具（create/append 即 dry-run/finalize CAS/status/apply 一次性 token），审批中心 `/change-plans`（Monaco diff、危险等级、侧边栏待审徽章），审批人≠创建人、apply 前逐步 RBAC 复查 |
| `d473ac7` | 顺序计划修复：delete 步骤目标由前序步骤创建时快照 NotFound 不再拒绝（降级 DRY_RUN_UNSUPPORTED） |
| `56803da` | **P3 资源同步引擎**：`pkg/syncengine` no-store reflector（手写分页 LIST + WATCH 续传 + 410 重建 + 退避），事件直接落 `sync_objects`/`sync_tasks`（内存零驻留），白名单默认关；泛型列表/详情/搜索优先走库、自动 fallback 直连；拓扑 API；集群连接预热 informer |
| `64f68c3` | 默认白名单补 replicasets/jobs（拓扑 owner 链曾断在第一层，浏览器实测发现） |
| `d5cae5a` | 聊天助手写路径接入审批（`aiChangePlanMode` 开关，kite-chat 计划审批即自动执行）；拓扑前端页 `/topology` |
| `0a74cf1` | `KUBOARD-GAP-ANALYSIS.md`（menu.yaml × OpenAPI × Kite 注册表深度对比，10 缺口排序）+ ReplicaSets 进工作负载侧边栏 |
| `6e2ea3b` | **G9** 同步引擎自观测（`/api/v1/admin/sync-engine/status` + restart-full-sync + 设置页面板）；**G4** 计划回滚（APPLIED → ROLLING_BACK → ROLLED_BACK，按快照逆序撤销，审批权限门槛） |
| `ec16e31` | **G1** Gateway API：sigs.k8s.io/gateway-api v1.3.0，gateways/httproutes/grpcroutes/referencegrants 注册进 scheme/registry/handlers/流量分组；NoMatch 泛型列表统一返回空（可选 CRD 缺失不再 500） |
| `dfd2905` | **G2** 快捷操作：`/api/v1/workload-ops/update-image` 与 `/restart`（deploy/sts/ds，非结构化对象 + update RBAC + 历史审计），Deployment/StatefulSet 详情页「编辑镜像」对话框 |

### 关键实现位置

- MCP：`pkg/mcpserver/{server,auth,tools,plan_tools}.go`；工具层复用 `pkg/ai`（`RunTool` 只读 / `ExecuteMutation` 写、`DryRunMutation`）
- 审批：`pkg/changeplan/{service,handler,chat_router,rollback}.go`，模型 `pkg/model/change_plan.go`
- 同步：`pkg/syncengine/{manager,reflector,store,status}.go`，模型 `pkg/model/sync.go`
- 快捷操作：`pkg/resources/workload_ops_handler.go`；前端 `ui/src/components/workload-image-dialog.tsx`、`ui/src/pages/{change-plans,topology}.tsx`

### 设计要点（不要回退）

- 计划审批三红线：apply 之外的路径不得写集群；approval_token 一次性消费；apply 前逐步 RBAC 复查。聊天计划（client=kite-chat）审批即自动执行（审批本身是闸门）
- 同步引擎 no-store：watch 事件直接落库，内存不驻留对象——几千 pod 规模下占用恒定，勿改回 informer 内存 Store
- 泛型列表 NoMatch 返回空列表是有意行为（可选 CRD 缺失的集群），不要改回 500

### 遗留（按 `KUBOARD-GAP-ANALYSIS.md` 优先级）

- G7 拓扑跨类关系边（service→pod selector、ingress→service，labels 已入库）
- G5 用户组管理 UI（后端已支持 group subject）、G6 审计策略、G10 Pod 调试容器注入（ephemeral containers）、G8 多集群聚合首页
- pods 的 `spec.nodeName` 内存索引仍在（节点页关联 pod 依赖），其内存卸载需先把该查询切到 sync 表
- 已知上游行为：角色变更会使已发 JWT 立即失效（改 RBAC 后被登出属预期，重新登录即可）

### 本地验证环境

`make dev` 起后端(8080)+vite(5173，若被占用顺延 5174)；登录 `admin` / `KiteDev#2026`
（dev.db 密码已重置）。真实集群 colima-xwssd 已导入；同步引擎/MCP 均可在
Settings → General 打开后按各卡片说明验证。
