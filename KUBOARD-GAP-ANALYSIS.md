# Kuboard 菜单与功能吸收分析

> 分析时间：2026-10-10
> 数据源：运行中的 Kuboard v4.3 容器（JAR 内 menu.yaml 菜单树 + OpenAPI 规范 109 端点/15 模块）× Kite `pkg/common/resource.go` 注册表（60+ 类型）与 `ui/src/lib/resource-catalog.ts`（68 个侧边栏条目）
> 原则：只对标语义，clean-room；已有的不重复吸收

## 一、总体结论

Kite 的资源类型覆盖度出乎意料地高：Kuboard 菜单里出现的 K8s 资源（Webhook、Quota、Lease、PriorityClass、FlowControl、DRA、CSR 等）Kite 注册表**全部都有**，只是大部分作为侧边栏自定义条目默认隐藏（Settings 里可通过侧边栏定制器开启）。

真正的缺口只有 **9 项**，按价值排序见第四节。

## 二、逐组对比

| Kuboard 菜单 | Kite 对应 | 状态 |
|---|---|---|
| 首页 Home | Overview 概览 | ✅ 已有（多集群聚合视图可增强，见 G8） |
| Agent 变更审批 | Change Plans（P2 已交付） | ✅ 已有 |
| 集群管理：Kubernetes 集群 | Settings → Cluster | ✅ 已有（IA 不同：Kite 放设置页，Kuboard 放侧边栏） |
| 集群管理：节点与运行时 | Nodes | ✅ 已有 |
| 集群管理：名称空间 | Namespaces | ✅ 已有 |
| 工作负载（动态子菜单） | Pods/Deployments/STS/DS/Jobs/CronJobs | ⚠️ G3：ReplicaSets 未进侧边栏 |
| 　└ Pod 日志/终端 | pod 详情内 WebSocket | ✅ 已有 |
| 服务与网络：服务与路由 | Services/Ingresses | ✅ 已有 |
| 服务与网络：**网关 Gateway API** | **无**（v0.16.0 上游删了 gateway 页面，注册表也无条目） | ❌ G1 |
| 服务与网络：networking.k8s.io | NetworkPolicies | ✅ 已有 |
| 服务与网络：discovery | EndpointSlices（other 组，默认隐藏） | ✅ 已有 |
| 配置与存储：配置中心 | ConfigMaps + Secrets | ✅ 已有（IA：合并 vs 分开） |
| 配置与存储：存储 | PVC/PV/StorageClasses | ✅ 已有 |
| 集群资源：集群 RBAC | 全套 RBAC 五件套 | ✅ 已有 |
| 集群资源：**Admission Webhook** | 注册表有（other 默认隐藏） | ✅ 数据有，开侧边栏即可 |
| 集群资源：**DRA 结构化资源** | 注册表有（ResourceSlice/ResourceClaim 等 5 类） | ✅ 数据有 |
| 集群资源：**Flow Control** | 注册表有（FlowSchema/PriorityLevelConfiguration） | ✅ 数据有 |
| 集群资源：scheduling/policy/coordination | PriorityClasses/ResourceQuotas+PDB/Leases 均有 | ✅ 数据有 |
| 定制资源 CRD | CRDs + 动态 CR 列表页 | ✅ 已有 |
| 应用管理：Helm Release | Helm Release + 详情/升级/回滚 | ✅ 已有 |
| 应用管理：市场浏览/仓库管理 | App Catalog + chart repos 管理 | ✅ 已有 |
| 应用管理：**常用操作三件套** | **无**：批量调镜像版本（CdApi）/ 批量导入 / 批量导出 | ❌ G2 |
| 运维与可观测：**资源全景图** | Topology（v2，仅 owner 链） | ⚠️ G7：缺跨类关系边 |
| 运维与可观测：Events | Events | ✅ 已有 |
| 运维与可观测：操作审计 | Settings → Audit | ✅ 已有 |
| 运维与可观测：**审计策略** | **无** | ❌ G6 |
| 系统管理：用户/角色 | Users/Roles | ✅ 已有 |
| 系统管理：**用户组** | 后端支持 group subject，无管理 UI | ⚠️ G5 |
| 系统管理：系统设置 | Settings | ✅ 已有 |
| 系统管理：License | — | 🚫 不吸收（商业逻辑，计划 non-goal） |
| Profile + MFA 绑定 | 用户菜单 MFA | ✅ 已有 |

## 三、菜单之外的后端能力（OpenAPI 挖掘）

| Kuboard 能力 | 端点证据 | Kite 状态 |
|---|---|---|
| **计划回滚** | `POST /change-plans/{id}/rollback`（按 before/after 快照回滚已 apply 的 step） | ❌ G4：Kite 计划只到 APPLIED/FAILED，无法回滚 |
| **计划删除/审计聚合** | DELETE plan、audit session-summary（MCP 会话聚合） | ⚠️ 部分 |
| **同步引擎自观测** | cache-status/count/tasks 日志/`restart-full-sync` | ❌ G9：Kite 有 sync_tasks 表，无查询 API/UI |
| **Pod 调试容器注入** | `POST /pods/ephemeral-containers`（kubectl debug） | ❌ G10：Kite 有 pod/node 终端，无临时调试容器注入 |
| Helm 离线包上传/values schema/managed resources | chart-upload 等 | ⚠️ 待核对 Kite Helm 页能力 |
| 集群 apiserver 连通性预检 | ping-apiserver | ⚠️ 小 |
| 菜单树管理/资源禁用 | k8s-resources/menu-tree + disabled | ✅ Kite 侧边栏定制器已覆盖 |
| Prometheus 地址自动发现 | cluster-cache/prometheus/location | ✅ Kite 已有 |
| MCP 订阅 SSE、OIDC 静默刷新、实例心跳锁 | — | 🚫 暂不吸收 |

## 四、缺口清单（按价值×成本排序）

| # | 缺口 | 价值 | 成本 | 说明 |
|---|---|---|---|---|
| G1 | **Gateway API 支持** | 高（K8s 1.35 网关生态主流，Ingress 替代品；Kuboard 有独立菜单） | 中：registry + catalog 各加 4~6 条（generic 页面自动生效），上游 v0.16.0 删除了旧实现需重建 | gateways/httproutes/grpcroutes/referencegrants |
| G2 | **常用操作三件套** | 高（Kuboard 招牌：批量改镜像版本是高频运维动作） | 中：update-image-tag 后端 + 批量选择 UI；导入导出（API 已支持多文档 YAML，缺页面） | 对应 CdApi restart-workload/update-image-tag |
| G4 | **计划回滚** | 高（审批流完整性：apply 失败/误操作可撤销；Kuboard 靠 before/after 快照回滚） | 中：rollback 端点 + 按 step 逆序还原；快照已存 | 补齐计划状态机 → ROLLED_BACK |
| G9 | **同步引擎自观测** | 中高（几千 pod 场景下用户需要看到同步健康度） | 低：sync_tasks/sync_objects 的查询 API + 设置页状态区块 + restart-full-sync | 表和数据都有了 |
| G3 | **ReplicaSets 侧边栏** | 中（数据已有，加条目） | 极低 | workloads 组 |
| G7 | **拓扑全景图增强** | 中（service→pod selector 边、ingress→service 边；Kuboard 全景图覆盖全资源） | 中：labels 已入库，需 selector 匹配 + 前端增强 | 现有 owner 链为骨架 |
| G6 | **审计策略** | 中低 | 中 | 先记录需求 |
| G5 | **用户组管理 UI** | 中低（后端已支持 group subject） | 低中 | OIDC 场景有用 |
| G10 | **Pod 调试容器** | 中低（临时容器注入 ephemeral-containers） | 低中 | kubectl debug 语义 |
| G8 | **多集群首页** | 低中 | 中 | Overview 聚合多集群卡片 |

## 五、不吸收清单

- **License 授权体系**：商业闭源逻辑，无吸收意义
- **套件市场/应用市场运营**：计划 non-goal
- **分布式实例锁**：Kite 单实例形态
- **密码 base64 传输 / JWT 私钥入库**：Kuboard 的安全反面教材，不学
- **配置中心合并视图**：纯 IA 偏好，Kite 分开更清晰
