# 生产环境安全基线与加固

本文档是 KubeOsprey 的生产环境安全基线，适用于 Helm 安装、独立 Kubernetes 清单和 Cluster Agent 部署。

KubeOsprey 是 Kubernetes 控制面。它可以创建高权限 Kubernetes 客户端、终端 Pod、exec 会话、代理请求，并执行 AI 辅助的资源变更。请把 KubeOsprey 视为集群管理边界来部署和审计。

## 生产环境基线

向用户开放 KubeOsprey 前，请确认：

- KubeOsprey、OAuth/OIDC 回调、Cluster Agent 连接和 WebSocket 流量均使用 HTTPS。
- 设置高熵的 `KITE_ENCRYPT_KEY` 并存储在外部 Secret 中，不要使用开发文档中的公开示例值。
- 如果需要重启后或多副本之间保持会话稳定，请显式设置 `JWT_SECRET`。
- 设置 `ANONYMOUS_USER_ENABLED=false`，并验证匿名访问已关闭。
- 将 `TRUSTED_PROXIES` 限制为实际的反向代理地址，不要信任任意客户端转发头。
- 使用外部 PostgreSQL/MySQL，或为 SQLite 配置持久化卷。
- 将 KubeOsprey、终端和 Cluster Agent 镜像固定到发布版本或 digest，生产环境不要使用 `latest`。
- 部署前检查每一条 `cluster-admin`、通配符 RBAC、privileged 容器、主机 namespace 和 hostPath 配置。
- 只启用运维人员确实需要的终端和 Cluster Agent 能力。
- 同时备份数据库和加密密钥。没有加密密钥的数据库备份无法恢复加密字段。

## 密钥与加密

数据库中的 `SecretString` 字段包括 kubeconfig、OAuth/LDAP 密钥、Helm 仓库密码、API Key、MFA Secret 和 Cluster Agent 私钥等凭据，它们使用 `KITE_ENCRYPT_KEY` 加密。

启动时会校验 `KITE_ENCRYPT_KEY`：密钥不能为空、长度至少 32 个字符、且不能等于内置的开发用默认值，否则服务拒绝启动。仅本地开发环境可以设置 `KITE_ALLOW_INSECURE_DEFAULTS=true` 跳过默认值检查。

`TRUSTED_PROXIES` 建议配置为实际反向代理的网段或 IP；设置为 `none` 表示忽略所有客户端转发头。

用户密码使用单向密码哈希保存，不能通过 `KITE_ENCRYPT_KEY` 恢复。

建议使用已有 Kubernetes Secret：

```yaml
secret:
  create: false
  existingSecret: kube-osprey-secrets
```

该 Secret 必须包含当前部署模式需要的配置项。恢复已有数据库时，必须保持原加密密钥不变。密钥轮换需要专门的迁移流程：使用旧密钥解密，再使用新密钥重加密。

不要把真实密钥写入 `values.yaml`、Git、Shell 历史、工单或生成的 Manifest。

## Kubernetes 权限

KubeOsprey 有应用层 RBAC，Kubernetes 还会独立检查 KubeOsprey Server 和 Cluster Agent 使用的 ServiceAccount 或 kubeconfig。最终权限受两套系统共同影响，但过大的 Kubernetes 身份仍会放大误操作或会话被攻破后的影响。

请分别检查：

- KubeOsprey Server 的 ServiceAccount 和 ClusterRole；
- Cluster Agent 的 ServiceAccount 和 ClusterRoleBinding；
- kubectl Terminal 的 ServiceAccount 和 ClusterRoleBinding；
- Node Terminal 的 host 访问、privileged、主机 namespace 和 hostPath；
- `imagePullSecrets` 以及终端 Pod 创建的 namespace。

生产环境避免使用以下通配符规则，除非该部署明确只用于管理员 Break-glass 场景：

```yaml
apiGroups: ["*"]
resources: ["*"]
verbs: ["*"]
```

Cluster Agent Manifest 支持权限档位：

- `readOnly`：全资源只读。
- `operator`：常见工作负载资源可读写（pods 含 exec/attach/portforward、services、configmaps、PVC、deployments、statefulsets、daemonsets、replicasets、jobs、cronjobs、ingresses、networkpolicies、PDB、HPA），并对 namespaces、nodes、PV、StorageClass、events、CRD、metrics、Secrets 只读；不授予 Secret 和 RBAC 的写权限。
- `admin`：绑定 `cluster-admin`（兼容旧的高信任行为）。
- `custom`：绑定指定的已有 ClusterRole。

默认档位为 `admin`（向后兼容）；生产环境建议使用 `operator` 或 `readOnly`。注意 `operator` 档位可以读 Secrets（KubeOsprey 控制台的 Secret 页面和 Helm Release 浏览依赖 Secret 数据）；如不可接受请使用 `custom`。

Kubernetes 中 ClusterRoleBinding 的 `roleRef` 不可修改。切换已有 Cluster Agent 的权限档位时，必须先删除 `kite-cluster-agent` ClusterRoleBinding，再应用新 Manifest。

## 终端与 exec 能力

Pod exec、kubectl Terminal 和 Node Terminal 都不是普通的只读功能：

- Pod exec 可以在工作负载容器内执行命令；
- kubectl Terminal 使用其 ServiceAccount 操作集群；
- Node Terminal 可以获得主机级能力。

应使用明确的管理员权限、短会话、自动清理和审计日志。除非确有需要，否则保持 kubectl 和 Node Terminal 关闭。请验证关闭后不会创建 ServiceAccount、Binding 或 Agent Pod。

## Prometheus 与原始 PromQL

PromQL Explorer 是集群级可观测性工具。只有允许查询配置的 Prometheus 数据源的用户才应获得该能力。Kubernetes namespace RBAC 不会自动过滤 Prometheus 查询结果。

如果 KubeOsprey 用作多租户平台，应在 Prometheus 层实现租户隔离，或改用审核过的查询模板。不要通过给任意 PromQL 盲目追加 namespace selector 来伪造数据隔离。

## 数据库与持久化

默认 SQLite 适合评估环境。没有持久化卷时，Pod 替换会丢失数据库。生产环境建议：

- 多副本和运维备份优先使用 PostgreSQL 或 MySQL；
- 使用 SQLite 时配置持久化卷，并使用适合单写卷的部署策略；
- 不仅测试备份创建，还要定期测试恢复；
- 将数据库、加密密钥和部署配置作为一个恢复集合备份。

## 审计策略

所有改变状态的console请求都会写入操作审计日志。策略在 **Settings → Audit** 中配置：

- **日志保留期**：操作日志保留的天数（1–3650，默认 30），超期条目自动清理。
- **跳过模块**：所选模块的变更不写入审计日志。请尽量保持该列表为空——跳过一个模块意味着它的变更不留任何审计痕迹。

若审计策略读取失败，审计按"失败放行"处理：变更仍会以默认保留期记录。

## Cluster Agent 检查清单

- 使用 HTTPS Server URL，并确保 Agent 信任对应证书；
- 将一次性 Agent Token 和生成的 YAML 当作凭据处理；
- 怀疑泄露后及时轮换或吊销 Agent Token；
- 应用 Manifest 前验证 Agent 镜像和 digest；
- 应用前检查 ServiceAccount、ClusterRoleBinding、Secret 和 Deployment；
- 切换权限档位时，先删除已有的 `kite-cluster-agent` ClusterRoleBinding（`roleRef` 无法原地更新）；
- 确保反向代理支持 WebSocket Upgrade 和长连接；
- 多副本部署前验证会话路由和共享状态。

## 验证命令

检查 Chart：

```bash
helm lint charts/kite
helm template kite charts/kite --namespace kite-system
bash scripts/validate-chart.sh
```

在测试集群中检查实际 Kubernetes 权限：

```bash
kubectl auth can-i --list \
  --as=system:serviceaccount:<namespace>:<service-account>

kubectl get clusterrolebinding <binding-name> -o yaml
kubectl get serviceaccount <service-account> -n <namespace> -o yaml
```

运行应用检查：

```bash
go test ./...
pnpm --dir ui exec tsc --noEmit -p tsconfig.app.json
```

请区分进程存活和应用就绪。进程可以运行，但数据库、加密密钥或集群配置可能已经不可用。

## 当前限制与后续路线

以下问题需要独立设计，不应直接粗暴收紧权限：

- Cluster Agent 当前使用注册的 Kubernetes 凭据转发请求，不会在 Kubernetes API 层模拟 KubeOsprey 用户身份；
- kubectl 和 Node Terminal 需要更强的 Break-glass 控制和更小权限的身份；
- 加密密钥轮换需要兼容已有 `SecretString` 的迁移流程；
- KubeOsprey 不会对原始 PromQL 做 namespace 改写，多租户隔离应由指标架构或受限查询模型提供；
- Passkey、Cluster Agent 会话和其他进程内状态在正式支持多副本前需要共享存储。
