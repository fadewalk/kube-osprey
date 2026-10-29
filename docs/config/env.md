# 环境变量

KubeOsprey 默认支持一些环境变量，来改变一些配置项的默认值。

::: tip 由 `KITE_*` 更名而来
环境变量前缀已由 `KITE_` 更名为 `KUBE_OSPREY_`（例如 `KITE_ENCRYPT_KEY` → `KUBE_OSPREY_ENCRYPT_KEY`）。旧名称仍然有效，但会打印弃用告警；请在方便时更新部署配置。
:::

- **PLUGIN_DIR**：已安装插件的文件目录，默认是相对于 KubeOsprey 工作目录的 `data/plugins`（官方容器镜像中为 `/app/data/plugins`）。Helm Chart 将其设为 `/data/plugins`，可复用 SQLite 的存储卷。目录与持久化配置见[插件存储](./chart-values#插件存储)。

- **KUBE_OSPREY_CONFIG_FILE**：配置文件路径。该功能仅适用于 KubeOsprey `v0.10.0` 及以上版本。设置后，KubeOsprey 从该文件加载集群、OAuth、LDAP、RBAC 和超级用户设置。详见[配置文件](/config/config-file)。
- **PLUGIN_DEV_URL**：开发插件的 manifest 地址，例如 `http://localhost:5174/plugin.json`。浏览器直接加载该地址，同 ID 时优先使用开发插件，不修改安装记录。不设置时仅加载已安装插件。Helm 部署可通过 `extraEnvs` 配置，详见[插件调试](/plugins/debugging)。
- **KUBE_OSPREY_USERNAME**：兼容旧配置的超级用户名环境变量。仅在未设置 `KUBE_OSPREY_CONFIG_FILE` 时，用于环境变量到数据库配置的迁移。
- **KUBE_OSPREY_PASSWORD**：兼容旧配置的超级用户密码环境变量。仅在未设置 `KUBE_OSPREY_CONFIG_FILE` 时，用于环境变量到数据库配置的迁移。
- **KUBECONFIG**：兼容旧配置的 kubeconfig 环境变量。仅在未设置 `KUBE_OSPREY_CONFIG_FILE` 时读取并导入集群配置。
- **ANONYMOUS_USER_ENABLED**：启用匿名用户访问，默认值为 `false`，当启用后所有访问将不再需要身份验证，并且默认拥有最高权限。

- **KUBE_OSPREY_BASE**：应用的部署基路径（如 `/kite`），用于在子路径下部署时的静态资源和路由前缀。
- **NAMESPACE**：Cluster Agent Pod 与终端等辅助 Pod 所在的命名空间，默认值为 `kube-system`。

- **JWT_SECRET**：用于签名和验证 JWT 的密钥
- **KUBE_OSPREY_ENCRYPT_KEY**：用于加密敏感数据的密钥，例如 OAuth client secret、kubeconfig 和 Cluster Agent 私钥。生产环境必须显式配置；恢复已有数据库时必须保持该密钥不变。
- **KUBE_OSPREY_ALLOW_INSECURE_DEFAULTS**：仅用于本地开发，设置为 `true` 后允许使用公开的开发环境加密密钥回退值。生产环境禁止设置。

- **HOST**: 用户 OAuth 2.0 授权回调地址生成，默认会从请求头获取，如果您发现结果不及预期可以手动配置此环境变量。

- **TRUSTED_PROXIES**：以逗号分隔的反向代理、Ingress 或负载均衡器 IP/CIDR 列表；只有这些直连 KubeOsprey 的上一跳才会被信任，KubeOsprey 才会读取其转发的 `X-Forwarded-For` / `X-Real-IP` 来判断客户端 IP。默认信任本地和常见私网网段（`127.0.0.0/8`、`10.0.0.0/8`、`172.16.0.0/12`、`192.168.0.0/16`、`::1`、`fc00::/7`），方便常见 Ingress 部署拿到真实用户 IP。生产环境建议配置为更窄的范围，例如 `TRUSTED_PROXIES=10.42.0.0/16,192.168.1.10`；如需忽略所有客户端转发头，可设置 `TRUSTED_PROXIES=none`。

- **DB_TYPE**：数据库类型，取值为 `sqlite`、`mysql` 或 `postgres`，默认值为 `sqlite`。
- **DB_DSN**：数据库连接串。SQLite 模式下为数据库文件路径，默认值为 `dev.db`；MySQL/PostgreSQL 模式下为完整的 DSN。
- **DISABLE_CACHE**：设置为 `true` 时禁用 Kubernetes 资源的客户端缓存（informer），所有读取直接访问 API Server。默认启用缓存。

- **KUBE_OSPREY_CACHE_SYNC_TIMEOUT**：启动时等待 Kubernetes 资源缓存同步完成的超时时间（秒），默认值为 `60`。大型或远程集群可适当调大。
- **KUBE_OSPREY_KUBE_API_QPS**：Kubernetes API 客户端的 QPS 限制，默认值为 `50`。
- **KUBE_OSPREY_KUBE_API_BURST**：Kubernetes API 客户端的突发请求上限，默认值为 `100`。

- **DISABLE_GZIP**：是否禁用响应的 GZIP 压缩，默认值为 `true`；设置为 `false` 时启用压缩。
- **DISABLE_VERSION_CHECK**：设置为 `true` 时关闭版本更新检查，默认值为 `false`。
- **CORS_ALLOWED_ORIGINS**：以逗号分隔的允许跨域来源列表，默认为空（不开启 CORS）。仅在需要让其他站点的前端直接调用 API 时配置。

- **KUBECTL_TERMINAL_IMAGE**: 用于 kubectl 终端的 Docker 镜像，默认值为 `zzde/kubectl:latest`。

- **NODE_TERMINAL_IMAGE**: 用于生成 Node Terminal Agent 的 Docker 镜像，默认值为 `busybox:latest`。

- **CLUSTER_AGENT_IMAGE**: 为 Cluster Agent 类型集群生成 Cluster Agent 部署清单时使用的 Docker 镜像，默认值为 `ghcr.io/kite-org/kite:latest`。

- **PORT**：KubeOsprey 运行的端口，默认值为 `8080`。

- **KUBE_OSPREY_MCP_ENABLED**：内置 MCP 服务（`POST /api/v1/mcp`）的初始默认值，默认为 `false`。该值仅在首次运行时写入通用设置，之后管理员可在 设置 → 通用 中调整。
- **KUBE_OSPREY_MCP_RATE_LIMIT_PER_MINUTE**：MCP 工具调用每用户每分钟的限流次数，默认值为 `60`。
