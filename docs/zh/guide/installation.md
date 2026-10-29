# 安装指南

本指南详细介绍如何在 Kubernetes 环境下安装 KubeOsprey。

## 前提条件

- 拥有集群管理员权限的 `kubectl`
- Helm v4，或支持 OCI chart 的 Helm v3.8+
- MySQL/PostgreSQL 数据库，或本地存储用于 sqlite

## 安装方式

### 方式一：Helm Chart（推荐）

使用 Helm 可灵活配置和升级 KubeOsprey：

```bash
helm install kube-osprey oci://ghcr.io/fadewalk/charts/kube-osprey -n kube-osprey-system --create-namespace
```

或锁定 Chart 版本以保证安装可复现：

```bash
helm install kube-osprey oci://ghcr.io/fadewalk/charts/kube-osprey --version <chart-version> -n kube-osprey-system --create-namespace
```

#### 自定义安装

可通过自定义 values 文件调整安装参数：

完整配置参考 [Chart Values](../config/chart-values)。

使用自定义值安装：

```bash
helm install kube-osprey oci://ghcr.io/fadewalk/charts/kube-osprey -n kube-osprey-system -f values.yaml
```

### 方式二：YAML 清单

如需快速部署，可直接应用官方安装 YAML：

::: warning
此方法不适合生产环境，因为他没有配置任何持久化相关内容，你需要手动挂载持久化卷并设置环境变量 `DB_DSN=/data/db.sqlite` 来确保数据不会丢失。或者也可以使用外部数据库。

参考 [持久化相关](../faq#持久化相关) 获取更多详情。
:::

```bash
kubectl apply -f https://raw.githubusercontent.com/fadewalk/kube-osprey/main/deploy/install.yaml
```

此方法将使用默认配置安装 KubeOsprey。如需高级定制，建议使用 Helm Chart。

## 访问 KubeOsprey

### 端口转发（测试环境）

测试期间可通过端口转发快速访问 KubeOsprey：

```bash
kubectl port-forward -n kube-osprey-system svc/kube-osprey 8080:8080
```

### LoadBalancer 服务

如集群支持 LoadBalancer，可直接暴露 KubeOsprey 服务：

```bash
kubectl patch svc kube-osprey -n kube-osprey-system -p '{"spec": {"type": "LoadBalancer"}}'
```

获取分配的 IP：

```bash
kubectl get svc kube-osprey -n kube-osprey-system
```

### Ingress（生产环境推荐）

生产环境建议通过 Ingress 控制器并启用 TLS 暴露 KubeOsprey：

::: warning
KubeOsprey 的日志和 Web 终端功能需支持 websocket。
部分 Ingress 控制器可能需额外配置以正确处理 websocket。
:::

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kube-osprey
  namespace: kube-osprey-system
spec:
  ingressClassName: nginx
  rules:
    - host: kube-osprey.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: kube-osprey
                port:
                  number: 8080
  tls:
    - hosts:
        - kube-osprey.example.com
      secretName: kube-osprey-tls
```

## 在子路径下部署（basePath）

如果您希望将 KubeOsprey 部署在一个子路径下，例如 `https://example.com/kube-osprey`，可以使用 Helm Chart 的 `basePath` 值来配置。

如何设置：

- 在 `values.yaml` 中：

```yaml
basePath: "/kube-osprey"
```

- 或使用 Helm CLI：

```fish
helm install kube-osprey oci://ghcr.io/fadewalk/charts/kube-osprey -n kube-osprey-system --create-namespace --set basePath="/kube-osprey"
```

说明：

- Ingress 配置：确保 Ingress 的 `paths` 与子路径一致，并使用合适的 `pathType`（例如 `Prefix`）。示例：

```yaml
ingress:
  enabled: true
  hosts:
    - host: kube-osprey.example.com
      paths:
        - path: /kube-osprey
          pathType: Prefix
```

- OAuth / 重定向：如果启用了 OAuth 或其他外部重定向，请在 OAuth 提供方中将重定向 URL 更新为包含子路径，例如 `https://kube-osprey.example.com/kube-osprey/oauth/callback`。

## 验证安装

安装完成后，可访问仪表盘验证 KubeOsprey 是否部署成功。预期界面如下：

::: tip
如需通过环境变量配置 KubeOsprey，请参考 [环境变量](../config/env)。
:::

![setup](../../screenshots/setup.png)

![setup](../../screenshots/setup2.png)

可根据页面提示完成集群设置。

## 卸载 KubeOsprey

### Helm 卸载

```bash
helm uninstall kube-osprey -n kube-osprey-system
```

### YAML 卸载

```bash
kubectl delete -f https://raw.githubusercontent.com/fadewalk/kube-osprey/main/deploy/install.yaml
```

## 后续步骤

KubeOsprey 安装完成后，您可以继续：

- [添加用户](../config/user-management)
- [配置 RBAC](../config/rbac-config)
- [配置 OAuth 认证](../config/oauth-setup)
- [设置 Prometheus 监控](../config/prometheus-setup)
