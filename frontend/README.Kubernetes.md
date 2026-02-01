# Frontend Kubernetes 部署（与 Backend 同集群）

本目录 `k8s/` 包含 Frontend（Next.js standalone）的 Deployment、Service、Ingress 与 ConfigMap，与 Backend 共用 **Minikube** 与 **nginx Ingress**。

## 前置条件

- 集群已启用 Ingress（`minikube addons enable ingress`）
- Backend 已部署（前端通过 `BACKEND_URL` 访问后端，ConfigMap 中为 `http://twixter-backend:8080`）。**必须设置 `BACKEND_URL`**，否则服务端请求（首页/探索等拉取 tags、feed）会走 `NEXT_PUBLIC_API_URL`（如 api.twixter.local），在 Pod 内无法解析，导致侧栏热门标签等只在进入 profile 等客户端请求的页面才有数据。

## 1. 构建镜像

**K8s 的 .env（Secret）只影响运行时**，不会改写已打包的客户端代码。**NEXT_PUBLIC_*** 在 `next build` 时被打进浏览器 bundle，所以**构建镜像时仍要传入**（`--build-arg`）。**BACKEND_URL** 仅服务端用，可从 K8s ConfigMap/Secret 在运行时注入，构建时可不传或传占位符。

**方式 A：从同一份 .env 读构建参数**（推荐，只维护 .env）：

```bash
# 使用 Minikube 内置 Docker
eval $(minikube docker-env)
set -a && . ./.env && set +a
docker build -t damonleelcx/twixter.store-frontend:latest \
  --build-arg NEXT_PUBLIC_API_URL="$NEXT_PUBLIC_API_URL" \
  --build-arg NEXT_PUBLIC_APP_URL="$NEXT_PUBLIC_APP_URL" \
  --build-arg NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY="$NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY" \
  --build-arg NEXT_PUBLIC_PAYPAL_CLIENT_ID="$NEXT_PUBLIC_PAYPAL_CLIENT_ID" \
  --build-arg BACKEND_URL="${BACKEND_URL:-http://twixter-backend:8080}" \
  .
```

**方式 B：命令行写死**（适合一次性或 CI 里用变量替换）：

```bash
eval $(minikube docker-env)
docker build -t damonleelcx/twixter.store-frontend:latest \
  --build-arg NEXT_PUBLIC_API_URL=http://api.twixter.local \
  --build-arg NEXT_PUBLIC_APP_URL=http://www.twixter.local \
  --build-arg NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_xxx \
  --build-arg NEXT_PUBLIC_PAYPAL_CLIENT_ID=xxx \
  --build-arg BACKEND_URL=http://twixter-backend:8080 \
  .
```

或使用 `build-and-push.sh` 并传入相应环境变量后推送到 Docker Hub。

## 2. 部署

**仅部署前端**（在 frontend 目录下）：

```bash
kubectl apply -k k8s/
```

**与后端一起从仓库根目录一次性部署**：

```bash
# 在仓库根目录
kubectl apply -k k8s/
```

## 3. 访问

- **端口转发**（不改 hosts）：
  ```bash
  kubectl port-forward svc/twixter-frontend 3000:3000
  ```
  浏览器访问 **http://localhost:3000**。

- **Ingress 域名**：在 hosts 中增加（将 `<MINIKUBE_IP>` 换为 `minikube ip` 输出）：
  ```
  <MINIKUBE_IP> www.twixter.local
  ```
  然后访问 **http://www.twixter.local**。

## 4. 配置说明

| 文件 | 说明 |
|------|------|
| `configmap.yaml` | 非敏感配置：`BACKEND_URL`（集群内后端地址）、`NODE_ENV`、`PORT` |
| `deployment.yaml` | Frontend 部署，从 ConfigMap + Secret（.env）注入环境变量；镜像需构建时传入 NEXT_PUBLIC_* |
| `service.yaml` | ClusterIP Service，端口 3000 |
| `ingress.yaml` | Ingress，主机名 `www.twixter.local` |

运行时变量由 ConfigMap（非敏感默认）与 Secret（从 `frontend/.env` 生成）共同提供，Secret 会覆盖同名键。
