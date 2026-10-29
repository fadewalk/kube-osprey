# 架构

KubeOsprey 以单个 Go 二进制交付，前端 React 应用直接内嵌其中。本页的图展示一次请求在这个二进制内部的完整路径、请求如何到达目标集群，以及 AI 助手、指标链路和构建流程在整体中的位置。

每张图都可交互。内嵌视图支持缩放、拖动和主题切换；点击 **全屏打开** 可使用引导视图逐条查看路径，并获得组件搜索和摘要卡片。

::: tip 建议阅读顺序
先看[系统总体架构](#系统总体架构)建立整体心智模型，再沿着[请求生命周期](#资源-api-请求生命周期)跟踪一次调用。其余几张图分别深入多集群接入、AI 助手、指标与构建，可按兴趣任意顺序阅读。
:::

## 系统总体架构

浏览器加载 React SPA，SPA 通过 REST、WebSocket 和 SSE 与 Gin HTTP 服务通信。每个 API 请求先经过 `RequireAuth` 与 RBAC 中间件，再到达 `pkg/resources` 中的资源处理器，处理器从 `ClusterManager` 获取对应集群的 `ClientSet`。集群既可以直连（in-cluster 或 kubeconfig），也可以通过[集群 Agent](./cluster-agent) 从隔离网络建立反向隧道接入。AI 助手、Prometheus 访问以及基于 GORM 的存储都挂在同一个服务上。

<DiagramFrame src="/diagrams/kube-osprey-architecture.html" title="KubeOsprey 系统总体架构" :height="520" />

鉴权由 KubeOsprey 自身数据库中的角色模型决定，而不是 Kubernetes RBAC。角色如何映射到资源和集群，请参阅 [RBAC 配置](/config/rbac-config)。

## 资源 API 请求生命周期

以 `GET /api/v1/pods?namespace=_all` 为例：请求使用 `auth_token` cookie 或 API Key 完成认证，通过带缓存的查询加载用户及其角色绑定，按资源动词完成校验后才进入处理器。读取优先走 Informer 缓存，未命中时回源 Kubernetes API。JWT 过期时，前端会静默调用 `POST /api/auth/refresh` 并重试原请求。

<DiagramFrame src="/diagrams/kube-osprey-request-sequence.html" title="资源 API 请求生命周期" :height="520" />

JWT 本身有效期 24 小时，cookie 有效期 48 小时，两者之差就是静默刷新能够成功的窗口。API Key 格式与刷新行为详见[认证](/api/authentication)。

## 集群客户端同步生命周期

`ClusterManager` 每分钟对全部集群记录执行一次 reconcile，配置变更后调用 `TriggerClusterSync` 也会立即触发。对每个集群，它根据配置的凭据来源构建 `ClientSet`、选择接入模式，并在可用时自动发现 Prometheus 地址。构建失败属于可恢复状态，会在下一轮重试；删除或禁用集群后，对应客户端随之移除。

<DiagramFrame src="/diagrams/kube-osprey-cluster-sync.html" title="集群客户端同步生命周期" :height="580" />

## cluster-agent 注册与反向隧道

对于服务端无法直接访问的集群，管理员以 Agent 模式添加集群，KubeOsprey 生成带权限档案（`readOnly`、`operator` 或 `custom`）的部署清单。Agent 就是同一个二进制以子命令方式运行。启动后它用 NaCl-box 加密凭据发起注册，服务端将注册 Token 与存储的哈希比对，校验通过后建立 `remotedialer` WebSocket 作为反向隧道，`ClientSet` 经由该隧道拨号访问集群 API。

<DiagramFrame src="/diagrams/kube-osprey-cluster-agent.html" title="cluster-agent 注册与反向隧道" :height="540" />

部署步骤与权限档案说明见 [KubeOsprey Cluster Agent](./cluster-agent)。

## AI 会话与工具调用

[AI 助手](./ai-assistant)通过 `POST /api/v1/ai/chat` 建立 SSE 流，并针对配置的 LLM 提供方运行 Agent 循环。模型提出的每次工具调用都要经过 `AuthorizeToolForUser`，它把调用映射到发起用户自身的 RBAC 权限，因此助手的能力永远不会超过使用它的人。变更类工具还会在对话中弹出确认卡片；确认后工具通过集群 `ClientSet` 执行，结果中的敏感字段被脱敏，变更记录写入[资源历史](./resource-history)。

<DiagramFrame src="/diagrams/kube-osprey-ai-toolcall.html" title="AI 会话与工具调用" :height="500" />

## 监控指标数据流

[监控](./monitoring)页面的指标来自集群内的 Prometheus，由 KubeOsprey 自动发现或按 [Prometheus 设置](/config/prometheus-setup)指定。查询按集群分别路由：集群内地址经 Kubernetes API proxy 转发，Agent 模式集群走隧道拨号，其余直连。`pkg/prometheus` 发出 `Query` 与 `QueryRange` 请求，`pkg/metrics` 在 `/api/v1/prometheus/*` 下暴露端点并做 RBAC 过滤；Prometheus 不可用时，Pod 实时指标回退到 `metrics-server`。

<DiagramFrame src="/diagrams/kube-osprey-metrics-dataflow.html" title="监控指标数据流" :height="470" />

## 构建与交付流水线

`make build` 先在 `ui/` 中执行 `pnpm build`，Vite 产物输出到 `static/`；`static.go` 用 `//go:embed` 嵌入该目录，`go build` 以 `CGO_ENABLED=0`、`-trimpath` 编译出 `kube-osprey` 二进制，版本信息通过 `ldflags` 注入。同一个二进制打包为容器镜像，并由 Helm chart 部署。`make dev` 会编译后端、以 `-v=5` 在后台运行，再启动 Vite 开发服务器，由它代理 `/api` 并提供热更新。

<DiagramFrame src="/diagrams/kube-osprey-build-pipeline.html" title="构建与交付流水线" :height="460" />

Helm 安装路径见[开始](./installation)。

## 范围与维护

这些图覆盖核心请求路径、集群接入、AI 助手、指标与构建。[告警中心](./alert-center)、[日志中心](./log-center)、[变更计划](./change-plans)和 [MCP 服务器](./mcp-server)等功能模块都运行在同一条 Gin → `ClusterManager` 路径上，各自的指南中有详细说明。

每张图都由 [`docs/diagrams/`](https://github.com/fadewalk/kube-osprey/tree/main/docs/diagrams) 中的规格文件经 [Archify](https://github.com/tt-a1i/archify) 生成。代码结构变化后，修改对应规格并重新生成 HTML 即可，该目录下的 README 列出了具体命令。
