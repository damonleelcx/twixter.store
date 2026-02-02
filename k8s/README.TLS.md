# k3s / Kubernetes 上为 Ingress 启用 HTTPS（解决 “Your connection isn’t secure”）

浏览器提示 **“Your connection to this site isn't secure”** 表示当前没有受信任的 TLS 证书（或证书未正确挂到 Ingress）。按下面步骤在 k3s 上用 **cert-manager + Let’s Encrypt** 自动签发并挂载证书。

## 前置条件

- 集群已安装 **nginx ingress**（`ingressClassName: nginx`）。
- 域名 **www.twixter.store**、**api.twixter.store** 的 DNS 已指向集群入口（LoadBalancer IP 或 NodePort 对应 IP），且 **80 / 443** 可从公网访问（Let’s Encrypt HTTP-01 需要访问你域名的 80 端口）。

## 1. 安装 cert-manager（若未安装）

k3s 默认不带 cert-manager，需要单独安装，例如：

```bash
# 添加 Helm 仓库并安装 cert-manager
helm repo add jetstack https://charts.jetstack.io
helm repo update
kubectl create namespace cert-manager
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --set installCRDs=true
```

或使用官方 manifest（见 [cert-manager 文档](https://cert-manager.io/docs/installation/)）：

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.4/cert-manager.yaml
```

确认 Pod 就绪：

```bash
kubectl get pods -n cert-manager
```

## 2. 创建 ClusterIssuer（Let’s Encrypt）

1. 编辑 `k8s/cert-manager-cluster-issuer.yaml`，将 **`YOUR_EMAIL@example.com`** 换成你的真实邮箱（用于 Let’s Encrypt 到期通知和账号关联）。
2. 应用 ClusterIssuer：

```bash
kubectl apply -f k8s/cert-manager-cluster-issuer.yaml
```

3. 确认已创建：

```bash
kubectl get clusterissuer letsencrypt-prod
```

## 3. 确保 Ingress 已引用 cert-manager

当前 Ingress 已配置：

- `cert-manager.io/cluster-issuer: "letsencrypt-prod"`
- `tls` 段中 `secretName: twixter-frontend-tls` / `twixter-backend-tls`

应用或更新 Ingress 后，cert-manager 会为每个 Ingress 自动创建 **Certificate** 和 **Challenge**，并通过 HTTP-01 向 Let’s Encrypt 完成校验。

## 4. 部署/更新应用与 Ingress

在仓库根目录执行：

```bash
kubectl apply -k k8s/
```

（若 ClusterIssuer 未纳入 kustomization，需单独执行：`kubectl apply -f k8s/cert-manager-cluster-issuer.yaml`。）

## 5. 检查证书是否签发成功

```bash
# 证书
kubectl get certificate -A
kubectl describe certificate twixter-frontend-tls -n default
kubectl describe certificate twixter-backend-tls -n default

# ACME 挑战
kubectl get challenges -A
```

当 Certificate 的 `READY` 为 `True` 时，对应 Secret（如 `twixter-frontend-tls`）中已有有效证书，nginx ingress 会用它做 TLS 终结，浏览器应不再提示 “connection isn’t secure”。

## 6. 若证书一直不 Ready

- **Challenge 失败**：确认 80 端口可从公网访问你的域名，且 nginx ingress 的 80 入口指向当前集群（无其他反向代理或防火墙拦截）。
- **限速**：Let’s Encrypt 生产环境有速率限制；调试时可先用 [staging 环境](https://cert-manager.io/docs/configuration/acme/#staging)（证书不被浏览器信任，仅用于验证流程）。
- **查看 cert-manager 日志**：

```bash
kubectl logs -n cert-manager -l app=cert-manager
kubectl logs -n cert-manager -l app=webhook
```

## 7. 未使用 cert-manager 时（自签名等）

若暂时不用 Let’s Encrypt，可手动创建 TLS Secret，并在 Ingress 的 `tls.secretName` 中引用该 Secret；浏览器会提示不受信任，需手动添加例外。生产环境建议使用 cert-manager + Let’s Encrypt。
