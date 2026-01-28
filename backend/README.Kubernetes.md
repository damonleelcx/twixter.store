# Backend Kubernetes 部署（Minikube + kubectl）

本目录 `k8s/` 包含 Backend 的 Deployment、Service、Ingress 与 Secret 配置，适用于 **Minikube** 与 **kubectl**。Postgres 使用**托管数据库**（Managed PostgreSQL）凭证，通过 Secret 注入。

## 前置条件

- 已安装 [Minikube](https://minikube.sigs.k8s.io/docs/start/) 与 [kubectl](https://kubernetes.io/docs/tasks/tools/)
- 已有托管 Postgres 连接信息（主机、用户、密码、库名、端口、SSL 模式）
- （可选）集群内或外部 Redis、Kafka，或后续再部署

## 1. 启动 Minikube 并启用 Ingress

```bash
minikube start
minikube addons enable ingress
```

## 2. 配置托管 Postgres 与其它凭证

编辑 `k8s/secret.yaml`，将占位符替换为实际值：

- **托管 Postgres**：从云厂商（AWS RDS、GCP Cloud SQL、Azure Database 等）获取的连接信息  
  - `DB_HOST`：实例主机名  
  - `DB_USER` / `DB_PASSWORD`：数据库用户与密码  
  - `DB_NAME`：数据库名  
  - `DB_PORT`：通常 5432  
  - `DB_SSLMODE`：托管库一般用 `require`
- 其它如 `AWS_*`、`STRIPE_*`、`KAFKA_BROKERS`、`REDIS_ADDR` 等按实际环境填写。

**注意**：不要将真实 `secret.yaml` 提交到版本库；可用 `kubectl create secret generic ...` 或 CI 注入，或使用 External Secrets 等方案管理。

## 3. 构建镜像并在 Minikube 内使用

在 Backend 项目根目录（含 Dockerfile 的目录）执行：

```bash
# 使用 Minikube 内置 Docker，构建的镜像直接可供集群使用
eval $(minikube docker-env)
docker build -t twixter-backend:latest .
```

`k8s/deployment.yaml` 中已使用 `image: twixter-backend:latest` 和 `imagePullPolicy: IfNotPresent`，因此会使用上述本地镜像。

## 4. 部署

```bash
# 在 backend 目录下
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

或一次性应用整个目录：

```bash
kubectl apply -f k8s/
```

## 5. 访问

- **Ingress 域名**：默认配置为 `api.twixter.local`。  
  在宿主机 hosts 中增加（将 `<MINIKUBE_IP>` 换为 `minikube ip` 输出）：

  ```
  <MINIKUBE_IP> api.twixter.local
  ```

  然后可通过：`http://api.twixter.local/health` 做健康检查。

- **仅集群内访问**：  
  `http://twixter-backend.default.svc.cluster.local:8080/health`

- **端口转发（不依赖 Ingress）**：  
  `kubectl port-forward svc/twixter-backend 8080:8080`  
  然后访问 `http://localhost:8080/health`。

## 6. 常用命令

```bash
# 查看 Pod / Service / Ingress
kubectl get pods,svc,ingress -l app=twixter-backend

# 查看 Backend 日志
kubectl logs -l app=twixter-backend -f

# 更新 Secret 后重启 Deployment
kubectl rollout restart deployment/twixter-backend
```

## 文件说明

| 文件 | 说明 |
|------|------|
| `secret.yaml` | 敏感配置：**托管 Postgres** 凭证、Redis、AWS、Stripe、Kafka 等 |
| `configmap.yaml` | 非敏感配置：TEMP_DIR、FFmpeg 路径等 |
| `deployment.yaml` | Backend 部署，从 Secret/ConfigMap 注入环境变量 |
| `service.yaml` | ClusterIP Service，端口 8080 |
| `ingress.yaml` | Ingress，配合 minikube ingress addon，主机名 `api.twixter.local` |

Postgres 使用托管实例时，不在此仓库中部署 Postgres Pod，仅通过 Secret 中的 `DB_*` 提供连接信息即可。
