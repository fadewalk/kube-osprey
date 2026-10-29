---
title: 托管 Kubernetes 集群配置
---

# 托管 Kubernetes 集群配置

## 问题说明

像 AKS (Azure Kubernetes Service)、EKS (Amazon Elastic Kubernetes Service) 等托管 Kubernetes 集群,默认的 kubeconfig 通常使用 `exec` 插件动态获取认证凭证。例如:

- **AKS** 使用 `kubelogin` 命令
- **EKS** 使用 `aws` CLI
- **GKE** 使用 `gcloud` 命令

这种认证方式在本地客户端环境中运行良好,但在 KubeOsprey 这样的服务端环境中会失败,因为:

1. 服务器上可能没有安装这些 CLI 工具
2. 即使安装了,服务器环境也可能没有相应的身份认证配置
3. 多租户场景下难以管理不同用户的凭证

### 使用 Service Account Token

为 KubeOsprey 创建一个专用的 Service Account,并使用其 token 进行认证。

KubeOsprey 提供了一个辅助创建的脚本

```sh
wget https://raw.githubusercontent.com/fadewalk/kube-osprey/main/scripts/generate-kube-osprey-kubeconfig.sh -O generate-kube-osprey-kubeconfig.sh
chmod +x generate-kube-osprey-kubeconfig.sh
./generate-kube-osprey-kubeconfig.sh
```

#### 步骤:

1. 创建 Service Account 和必要的 RBAC 权限:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-osprey-admin
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kube-osprey-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: kube-osprey-admin
    namespace: kube-system
```

2. 创建 Long-lived Token Secret (Kubernetes 1.24+):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kube-osprey-admin-token
  namespace: kube-system
  annotations:
    kubernetes.io/service-account.name: kube-osprey-admin
type: kubernetes.io/service-account-token
```

3. 获取 token 和集群信息:

```bash
# 获取 token
TOKEN=$(kubectl get secret kube-osprey-admin-token -n kube-system -o jsonpath='{.data.token}' | base64 -d)

# 获取 CA 证书
CA_CERT=$(kubectl get secret kube-osprey-admin-token -n kube-system -o jsonpath='{.data.ca\.crt}')

# 获取 API Server 地址
API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
```

4. 生成 kubeconfig:

```bash
cat > kube-osprey-kubeconfig.yaml <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: ${CA_CERT}
    server: ${API_SERVER}
  name: kube-osprey-cluster
contexts:
- context:
    cluster: kube-osprey-cluster
    user: kube-osprey-admin
  name: kube-osprey-context
current-context: kube-osprey-context
users:
- name: kube-osprey-admin
  user:
    token: ${TOKEN}
EOF
```

## 相关文档

- [Kubernetes Service Account Tokens](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/)
- [AKS Authentication](https://learn.microsoft.com/en-us/azure/aks/control-kubeconfig-access)
- [EKS Authentication](https://docs.aws.amazon.com/eks/latest/userguide/cluster-auth.html)
- [GKE Authentication](https://cloud.google.com/kubernetes-engine/docs/how-to/api-server-authentication)
