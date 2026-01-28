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

**方式 A：用 .env 生成 Secret（推荐）**

在 `backend` 目录下准备 `.env` 文件（可复制 `.env.example` 后填写），变量名与 `k8s/secret.yaml` 中的键一致（如 `DB_HOST`、`DB_USER`、`AWS_ACCESS_KEY_ID` 等）。部署时使用 Kustomize，会从 `.env` 自动生成 Secret：

- `.env` 不要提交到 Git（应在 `.gitignore` 中）。
- 部署用 **`kubectl apply -k k8s/`**（在 backend 目录下执行），不要用 `kubectl apply -f k8s/`。

**方式 B：手写 secret.yaml**

编辑 `k8s/secret.yaml`，将占位符替换为实际值。若使用此方式，需在 `k8s/kustomization.yaml` 中恢复 `secret.yaml` 到 `resources`，并注释掉或删除 `secretGenerator` 段，然后用 **`kubectl apply -f k8s/`** 部署。

变量说明（两种方式相同）：

- **托管 Postgres**：从云厂商（AWS RDS、GCP Cloud SQL、Azure Database 等）获取的连接信息  
  - `DB_HOST`：实例主机名  
  - `DB_USER` / `DB_PASSWORD`：数据库用户与密码  
  - `DB_NAME`：数据库名  
  - `DB_PORT`：通常 5432  
  - `DB_SSLMODE`：托管库一般用 `require`
- 其它如 `AWS_*`、`STRIPE_*`、`KAFKA_BROKERS`、`REDIS_ADDR` 等按实际环境填写。

**注意**：不要将真实凭证提交到版本库；用 .env 时保证其在 `.gitignore` 中，或使用 External Secrets 等方案管理。

## 3. 构建镜像并在 Minikube 内使用

**方式 A：从 Docker Hub 拉取**

`k8s/deployment.yaml` 使用 `image: damonleelcx/twixter.store-backend:latest`，若已推送至 Docker Hub，可直接部署（无需本地构建）。

**方式 B：本地构建**

在 Backend 项目根目录（含 Dockerfile 的目录）执行：

```bash
# 使用 Minikube 内置 Docker，构建的镜像直接可供集群使用
eval $(minikube docker-env)
docker build -t damonleelcx/twixter.store-backend:latest .
```

或使用脚本构建并推送到 Docker Hub：`./build-and-push.sh [tag]`（默认 tag 为 `latest`）。

## 4. 部署

**若使用 .env + Kustomize（方式 A）**，在 backend 目录下执行：

```bash
kubectl apply -k k8s/
```

**若使用 secret.yaml（方式 B）**，可逐文件或一次性应用：

```bash
# 在 backend 目录下
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

或一次性：`kubectl apply -f k8s/`

## 5. 访问

- **本地用 localhost 测试（推荐）**：不改 hosts、不依赖 Ingress，在终端执行：
  ```bash
  kubectl port-forward svc/twixter-backend 8080:8080
  ```
  保持该终端运行，在浏览器或前端访问 **`http://localhost:8080/health`** 等接口即可。Ctrl+C 结束端口转发。

- **Ingress 域名**：若希望通过域名 `api.twixter.local` 访问，默认配置为该主机名。  
  在宿主机 hosts 中增加（将 `<MINIKUBE_IP>` 换为 `minikube ip` 输出）：

  ```
  <MINIKUBE_IP> api.twixter.local
  ```

  **Windows**：hosts 文件在 `C:\Windows\System32\drivers\etc\hosts`。以**管理员身份**打开记事本，用“打开”打开该文件，在末尾添加上述一行（将 `<MINIKUBE_IP>` 换成 `minikube ip` 的输出，如 `192.168.49.2 api.twixter.local`），保存。

  **Linux**：hosts 文件在 `/etc/hosts`。用 sudo 编辑，例如：`sudo nano /etc/hosts` 或 `sudo vim /etc/hosts`，在末尾添加上述一行（将 `<MINIKUBE_IP>` 换成 `minikube ip` 的输出），保存退出。

  然后可通过：`http://api.twixter.local/health` 做健康检查。

- **仅集群内访问**：  
  `http://twixter-backend.default.svc.cluster.local:8080/health`

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
| `secret.yaml` | 敏感配置模板（可选）：与 .env 键一致时可手填；用 .env 时由 Kustomize `secretGenerator` 从 `.env` 生成 Secret |
| `configmap.yaml` | 非敏感配置：TEMP_DIR、FFmpeg 路径等 |
| `deployment.yaml` | Backend 部署，从 Secret/ConfigMap 注入环境变量 |
| `service.yaml` | ClusterIP Service，端口 8080 |
| `ingress.yaml` | Ingress，配合 minikube ingress addon，主机名 `api.twixter.local` |

Postgres 使用托管实例时，不在此仓库中部署 Postgres Pod，仅通过 Secret 中的 `DB_*` 提供连接信息即可。
