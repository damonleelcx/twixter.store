# Twixter.Store

Digital content platform with subscriptions, credits, and video processing.

---

## Prerequisites

- **Go** — backend runtime
- **Kafka** — message broker (e.g. for video processing)
- **Node.js** — frontend (see `frontend/`)

---

## Getting Started

### 1. Start Zookeeper

Kafka depends on Zookeeper. In a terminal, from your Kafka install directory:

```bash
cd path\to\kafka
zookeeper-server-start.bat config\zookeeper.properties
```

Use your actual Kafka path (e.g. `C:\Users\damon\kafka`) in place of `path\to\kafka`.

### 2. Start Kafka

In a **second** terminal, from the same Kafka directory:

```bash
cd path\to\kafka
kafka-server-start.bat config\server.properties
```

### 3. Run the backend

From the project root:

```bash
cd backend
go run main.go
```

---

## Development Notes

- **Access control:** Validate access by **permissions only**, not by account type.

---

---

## 用 Kubernetes 同时跑 Backend 和 Frontend（Minikube）

### 前置

- 已安装 [Minikube](https://minikube.sigs.k8s.io/docs/start/) 与 [kubectl](https://kubernetes.io/docs/tasks/tools/)
- 已有托管 Postgres 等后端配置（见 `backend/README.Kubernetes.md`）
- 已准备 `backend/k8s/.env`（用于生成后端 Secret；可从 `backend/.env` 复制）；若前端 Pod 需从 Secret 注入变量，则准备 `frontend/k8s/.env`（可从 `frontend/.env` 复制）

### 步骤 1：启动 Minikube 并启用 Ingress

```bash
minikube start
minikube addons enable ingress
```

### 步骤 2：获取 Minikube IP 并写入 hosts（前后端共用）

在终端执行：

```bash
minikube ip
```

记下输出的 IP（例如 `192.168.49.2`）。在**本机 hosts** 里为前后端各加一行，把 `<MINIKUBE_IP>` 换成上面得到的 IP：

| 主机名 | 用途 |
|------|------|
| `<MINIKUBE_IP>/127.0.0.1 api.twixter.local` | Backend（Ingress） |
| `<MINIKUBE_IP>/127.0.0.1 www.twixter.local` | Frontend（Ingress） |

**说明：** `*.twixter.local` 是本地测试用的主机名，**不需要购买或拥有这些域名**。`.local` 常用于本机/局域网；把上面两行写进 hosts 后，只有你这台电脑会把这两个名字解析到 Minikube IP，不会走公网 DNS。

**Windows**：用**管理员身份**打开记事本 → 打开 `C:\Windows\System32\drivers\etc\hosts`，在末尾添加：

```
<MINIKUBE_IP>/127.0.0.1 api.twixter.local
<MINIKUBE_IP>/127.0.0.1 www.twixter.local
```

保存。

**Linux / macOS**：编辑 `/etc/hosts`，例如：

```bash
sudo nano /etc/hosts
```

在末尾添加上述两行后保存。

### 步骤 3：构建镜像（使用 Minikube 内置 Docker）

先让当前终端使用 Minikube 内的 Docker（构建的镜像会直接进 Minikube）：

| 终端 | 命令 |
|------|------|
| **Bash / Git Bash** | `eval $(minikube docker-env)` |
| **PowerShell** | `minikube docker-env --shell powershell | Invoke-Expression` |
| **CMD** | `for /f "tokens=*" %i in ('minikube docker-env --shell cmd') do %i`（在 .bat 里用 `%%i`） |

然后执行下面的构建（以下以 Bash 为例；Windows 下用 PowerShell 或 CMD 时，多行可改成一行或用 `^` 续行）。

**环境变量 / .env 说明**  
- **Backend**：镜像构建不依赖 .env；**部署**（步骤 4）时需在 `backend/k8s/` 下存在 `.env`（可从 `backend/.env` 复制），用于 k8s 生成 Secret（见根目录 `k8s/` 及 `backend/k8s/kustomization.yaml`）。  
- **Frontend**：构建时需传入 `NEXT_PUBLIC_*` 等变量；若在 `frontend/` 下使用 `.env`，建议先加载再执行 `docker build`，这样 Stripe/PayPal 等 key 会通过 `--build-arg` 传入镜像。

**Backend**（在项目根目录下）：

```bash
eval $(minikube docker-env)
cd backend
# 构建不依赖 .env；部署阶段会用到 backend/k8s/.env 生成 Secret
docker build -t damonleelcx/twixter.store-backend:latest .
cd ..
```

**Windows (PowerShell) — Backend**（在项目根目录下）：

```powershell
minikube docker-env --shell powershell | Invoke-Expression
cd backend
# 构建不依赖 .env；部署阶段会用到 backend/k8s/.env 生成 Secret
docker build -t damonleelcx/twixter.store-backend:latest .
cd ..
```

**Windows (CMD) — Backend**（在项目根目录下；若写在 .bat 里，将 `%i` 改为 `%%i`）：

```cmd
for /f "tokens=*" %i in ('minikube docker-env --shell cmd') do %i
cd backend
REM 构建不依赖 .env；部署阶段会用到 backend/k8s/.env 生成 Secret
docker build -t damonleelcx/twixter.store-backend:latest .
cd ..
```

**Frontend**（在项目根目录下）：  
前端镜像构建时需传入浏览器访问的 API/APP 地址（用刚才在 hosts 里配的域名，不要写死 Minikube IP，这样换 IP 不用重做镜像）。若使用 `frontend/k8s/.env`，先加载再构建：

```bash
eval $(minikube docker-env)
cd frontend
# 若有 k8s/.env，先加载以便 NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY 等传入构建
[ -f k8s/.env ] && set -a && . ./k8s/.env && set +a
docker build -t damonleelcx/twixter.store-frontend:latest \
  --build-arg NEXT_PUBLIC_API_URL="http://api.twixter.local" \
  --build-arg NEXT_PUBLIC_APP_URL="http://www.twixter.local" \
  --build-arg NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY="${NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY:-}" \
  --build-arg NEXT_PUBLIC_PAYPAL_CLIENT_ID="${NEXT_PUBLIC_PAYPAL_CLIENT_ID:-}" \
  --build-arg BACKEND_URL="http://twixter-backend:8080" \
  .
cd ..
```

**Windows (PowerShell) — Frontend**（在项目根目录下；若有 `frontend/k8s/.env` 会先加载到当前进程）：

```powershell
minikube docker-env --shell powershell | Invoke-Expression
cd frontend
# 若有 k8s/.env，加载到当前进程以便 NEXT_PUBLIC_* 等传入构建
if (Test-Path k8s/.env) { Get-Content k8s/.env | Where-Object { $_ -notmatch '^\s*#' -and $_ -match '=' } | ForEach-Object { $p = $_ -split '=',2; Set-Item -Path "Env:$($p[0].Trim())" -Value $p[1].Trim() } }
docker build -t damonleelcx/twixter.store-frontend:latest `
  --build-arg NEXT_PUBLIC_API_URL="http://api.twixter.local" `
  --build-arg NEXT_PUBLIC_APP_URL="http://www.twixter.local" `
  --build-arg NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY="$env:NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY" `
  --build-arg NEXT_PUBLIC_PAYPAL_CLIENT_ID="$env:NEXT_PUBLIC_PAYPAL_CLIENT_ID" `
  --build-arg BACKEND_URL="http://twixter-backend:8080" `
  .
cd ..
```

**Windows (CMD) — Frontend**（在项目根目录下；若写在 .bat 里，将 `%a` `%b` 改为 `%%a` `%%b`）：

```cmd
for /f "tokens=*" %i in ('minikube docker-env --shell cmd') do %i
cd frontend
REM 若有 frontend/k8s/.env：把文件里的 KEY=VALUE 读进当前 CMD，下面 docker build 的 %NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY% 等会用到；没有可跳过本行
if exist k8s\.env for /f "usebackq eol=# tokens=1* delims==" %a in ("k8s\.env") do set "%~a=%~b"
docker build -t damonleelcx/twixter.store-frontend:latest ^
  --build-arg NEXT_PUBLIC_API_URL=http://api.twixter.local ^
  --build-arg NEXT_PUBLIC_APP_URL=http://www.twixter.local ^
  --build-arg NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=%NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY% ^
  --build-arg NEXT_PUBLIC_PAYPAL_CLIENT_ID=%NEXT_PUBLIC_PAYPAL_CLIENT_ID% ^
  --build-arg BACKEND_URL=http://twixter-backend:8080 ^
  .
cd ..
```

说明：`NEXT_PUBLIC_API_URL` / `NEXT_PUBLIC_APP_URL` 使用 `api.twixter.local` 和 `www.twixter.local`，本机通过 hosts 把这两个域名解析到 Minikube IP，因此**无需在构建参数里写 Minikube IP**；Minikube IP 只用在 hosts 里。  
**Minikube 构建时**：上面各段里的 `docker build` 已写死 API/APP/BACKEND 为 Minikube 用址（`api.twixter.local`、`www.twixter.local`、`twixter-backend:8080`），不会从 `frontend/k8s/.env` 读这些变量；`frontend/k8s/.env` 只用于传入 Stripe/PayPal 的 key。若 `frontend/k8s/.env` 里写了 `localhost:8080` 等，会被忽略，镜像仍按命令里的地址构建。

### 步骤 4：一次性部署前后端

部署前请确保 **backend/k8s/.env** 已存在（可从 `backend/.env` 复制）；前端需准备 **frontend/k8s/.env**（Stripe/PayPal 等 key，Kustomize 会据此生成 Secret）。前端构建时从 `frontend/k8s/.env` 加载并传入 Stripe/PayPal 等，镜像内已有；Pod 运行时还可通过该 Secret 再注入变量。

在**仓库根目录**执行：

```bash
kubectl apply -k k8s/
```

### 步骤 5：访问

- **Backend**：浏览器打开 `http://api.twixter.local/health`（会解析到 Minikube IP）
- **Frontend**：浏览器打开 `http://www.twixter.local`（会解析到 Minikube IP）

若 Minikube IP 变了，只需改 hosts 里上述两行的 IP，无需重新构建镜像。

### 修改代码后如何刷新 K8s（Backend / Frontend）

改完 **backend** 或 **frontend** 代码后，需要重新构建镜像并让集群用新镜像跑：

1. **重新构建镜像**（二选一）  
   - **Minikube 本地**：先 `eval $(minikube docker-env)`（Windows PowerShell：`minikube docker-env --shell powershell | Invoke-Expression`），再在对应目录构建：
     - Backend：`cd backend && docker build -t damonleelcx/twixter.store-backend:latest . && cd ..`
     - Frontend：`cd frontend && docker build -t damonleelcx/twixter.store-frontend:latest --build-arg NEXT_PUBLIC_API_URL=http://api.twixter.local --build-arg NEXT_PUBLIC_APP_URL=http://www.twixter.local --build-arg BACKEND_URL=http://twixter-backend:8080 . && cd ..`（按需加 `--build-arg`，与步骤 3 一致）
   - **推送到 Docker Hub**：在 `backend` 或 `frontend` 目录执行 `./build-and-push.sh`（或 `./build-and-push.sh <tag>`），集群从 Hub 拉取时需下一步强制重启。
2. **让 K8s 用新镜像**（必须做，否则会继续用旧 Pod）  
   在**仓库根目录**执行：
   - 只改了 backend：`kubectl rollout restart deployment/twixter-backend`
   - 只改了 frontend：`kubectl rollout restart deployment/twixter-frontend`
   - 两个都改了：`kubectl rollout restart deployment/twixter-backend deployment/twixter-frontend`
3. **只改了配置（ConfigMap/Secret）或 YAML，没改代码**  
   应用配置并重启对应部署即可：
   ```bash
   kubectl apply -k k8s/
   kubectl rollout restart deployment/twixter-backend deployment/twixter-frontend
   ```

说明：当前部署使用 `image: ...:latest` 且 `imagePullPolicy: IfNotPresent`，**仅执行 `kubectl apply -k k8s/` 不会拉新镜像**；必须执行 `kubectl rollout restart deployment/...` 才会重建 Pod。若镜像在 Docker Hub 更新过，节点会按策略拉取（若仍用旧缓存，可改为 `imagePullPolicy: Always` 或每次构建打新 tag）。

### 步骤 6：无法访问时的故障排查（no response）

若 `http://api.twixter.local/health` 或 `http://www.twixter.local` 无响应，按下面顺序检查。

**1. 确认 hosts 已配置**

- 执行 `minikube ip` 得到当前 Minikube IP（如 `192.168.49.2`）。
- 在本机 hosts 中必须有（把 `<MINIKUBE_IP>` 换成实际 IP）：
  - `<MINIKUBE_IP>/127.0.0.1 api.twixter.local`
  - `<MINIKUBE_IP>/127.0.0.1 www.twixter.local`
- Windows：`C:\Windows\System32\drivers\etc\hosts`（需管理员权限编辑）。
- 在终端验证：`ping api.twixter.local` 应解析到 Minikube IP。

**1.1 ping 超时、主机无法访问 Minikube IP（常见于 Windows）**

若 `ping api.twixter.local` 能解析到 `192.168.49.2` 但 **Request timed out / 100% loss**，说明 hosts 正确，但本机到 Minikube 虚拟机的网络不通（Windows + Docker 驱动下很常见）。

- **先试浏览器**：有时只是 ICMP 被拦，TCP 可用。直接打开 `http://api.twixter.local/health` 和 `http://www.twixter.local`，若能打开可忽略 ping。
- **开 Minikube 隧道（推荐）**：以**管理员身份**打开 PowerShell 或 CMD，执行后保持窗口不关：
  ```bash
  minikube tunnel
  ```
  **Windows（CMD / PowerShell）**：同上，直接执行 `minikube tunnel`。
  - **“卡住”是正常的**：该进程会一直运行，不会退出。**不要关这个终端**，另开浏览器访问 `http://api.twixter.local/health` 和 `http://www.twixter.local` 即可。
  - **如何判断隧道已开**：若终端里出现 **“Tunnel successfully started”** 以及 **“Starting tunnel for service twixter-backend-ingress”**、**“Starting tunnel for service twixter-frontend-ingress”**，说明隧道已成功启动；进程一直挂着不退出是预期行为，保持该窗口不关即可。
  - 若出现 **“Access to ports below 1024 may fail on Windows with OpenSSH clients older than v8.1”**：Ingress 使用 80 端口，在旧版 OpenSSH 的 Windows 上隧道可能无法转发 80 端口，此时通过 `http://www.twixter.local` 仍可能无响应。**先试浏览器**访问上述两个地址；若仍打不开，请**关闭 tunnel 窗口**，改用下面的**端口转发**方式访问（不依赖 Ingress）：`kubectl port-forward svc/twixter-frontend 3000:3000` 后访问 `http://localhost:3000`，后端同理 `kubectl port-forward svc/twixter-backend 8080:8080` 后访问 `http://localhost:8080/health`。
- **仍不通时改用端口转发**：不用域名，用本机端口访问。需开两个终端，分别执行：
  ```bash
  kubectl port-forward svc/twixter-backend 8080:8080
  kubectl port-forward svc/twixter-frontend 3000:3000
  ```
  **Windows（CMD / PowerShell）**：同上；终端 1 运行 `kubectl port-forward svc/twixter-backend 8080:8080`，终端 2 运行 `kubectl port-forward svc/twixter-frontend 3000:3000`。
  后端：`http://localhost:8080/health`，前端：`http://localhost:3000`。
- **长期方案**：若希望本机直接访问 Minikube IP，可改用能桥接到本机网段的驱动，例如：
  ```bash
  minikube stop
  minikube start --driver=hyperv
  ```
  **Windows（CMD / PowerShell）**：同上，依次执行 `minikube stop`、`minikube start --driver=hyperv`。
  然后重新执行 `minikube ip` 并更新 hosts。Hyper-V 需已启用。

**2. 确认 Ingress 已启用**

```bash
minikube addons list | grep ingress
# ingress 应为 enabled
minikube addons enable ingress   # 若未启用则执行
```

**Windows（CMD）**：

```cmd
minikube addons list | findstr ingress
REM ingress 应为 enabled
minikube addons enable ingress
```

**Windows（PowerShell）**：

```powershell
minikube addons list | Select-String ingress
# ingress 应为 enabled
minikube addons enable ingress
```

**3. 确认 Ingress Controller 在运行**

```bash
kubectl get pods -n ingress-nginx
# 应有 ingress-nginx-controller 等 Pod 且为 Running
```

**Windows（CMD / PowerShell）**：同上，执行 `kubectl get pods -n ingress-nginx`。

**4. 确认前后端 Pod 与 Service 正常**

```bash
kubectl get pods
# twixter-backend-* 与 twixter-frontend-* 应为 Running，READY 1/1
kubectl get svc
# twixter-backend、twixter-frontend 应存在
kubectl get ingress
# 两个 Ingress 的 ADDRESS 应有值（或为 minikube ip）
```

**Windows（CMD / PowerShell）**：同上，依次执行 `kubectl get pods`、`kubectl get svc`、`kubectl get ingress`。

若 Pod 为 `ImagePullBackOff` / `ErrImagePull`，请用 Minikube 内置 Docker 构建镜像（步骤 3）并确认镜像名为 `damonleelcx/twixter.store-backend:latest` 与 `damonleelcx/twixter.store-frontend:latest`。

**4.1 Backend CrashLoopBackOff**

若 `twixter-backend-*` 为 **CrashLoopBackOff**，先看崩溃原因：

```bash
kubectl logs deployment/twixter-backend
```

或查看上一个崩溃实例：`kubectl logs twixter-backend-54f6fb9f4c-tz57v --previous`（把 Pod 名换成当前列表里的）。

**4.2 www.twixter.local 无响应（前端）、后端正常**

若 `http://api.twixter.local/health` 能打开，但 `http://www.twixter.local` 无响应，按下面检查：

1. **前端 Pod 是否在跑**：`kubectl get pods -l app=twixter-frontend`  
   - 若为 `Running` 且 `READY 1/1`，继续下一步。  
   - 若为 `CrashLoopBackOff` 或 `Pending`：`kubectl logs deployment/twixter-frontend`（或 `kubectl describe pod <frontend-pod名>`）看原因；常见为镜像未构建或未拉取（`ImagePullBackOff`），需在 Minikube 内构建前端镜像并确认名为 `damonleelcx/twixter.store-frontend:latest`。

2. **hosts 是否包含 www**：本机 hosts 里除 `api.twixter.local` 外，必须有 `<MINIKUBE_IP>/127.0.0.1 www.twixter.local`（`minikube ip` 得到 IP）。  
   - 终端执行：`ping www.twixter.local`，应解析到 Minikube IP。

3. **Windows 是否开了 Minikube 隧道**：Ingress 使用 80 端口，在 Windows 上通常需要**以管理员身份**开一个终端并执行 `minikube tunnel`，保持不关。否则本机访问 `www.twixter.local` 可能无响应。  
   - 若 tunnel 报错或无法用，改用端口转发：`kubectl port-forward svc/twixter-frontend 3000:3000`，浏览器访问 `http://localhost:3000`。

4. **Ingress 是否生效**：`kubectl get ingress` 中应有 `twixter-frontend-ingress`，且 `HOSTS` 为 `www.twixter.local`。  
   - 快速区分问题：若 `kubectl port-forward svc/twixter-frontend 3000:3000` 后访问 `http://localhost:3000` 正常，则问题在 **hosts 或 Ingress 或 minikube tunnel**；若 port-forward 也无响应，则问题在 **前端 Pod/应用**。

**常见原因与处理：**

1. **Database initialization failed**  
   - 说明集群内无法连接 Postgres。Secret 来自 `backend/k8s/.env`（Kustomize 的 secretGenerator）。  
   - 检查 `backend/k8s/.env` 中 `DB_HOST`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`、`DB_PORT`、`DB_SSLMODE` 是否为**托管 Postgres** 的真实值，且从本机/集群能访问该主机（防火墙、安全组、VPC）。  
   - 修改后重新部署：`kubectl apply -k k8s/`，再 `kubectl rollout restart deployment/twixter-backend`。

   **使用本机 Postgres（Minikube）**：后端在 Pod 里跑，Pod 里的 `localhost` / `127.0.0.1` 指向 Pod 自己，**不是你的电脑**。若 Postgres 装在本机，在 `backend/k8s/.env` 里要把 `DB_HOST` 写成集群能解析到本机的地址，不要用 `localhost` 或 `127.0.0.1`：
   - **Minikube**：`DB_HOST=host.minikube.internal`（Minikube 提供的指向宿主机的主机名）。
   - **Docker Desktop K8s**：`DB_HOST=host.docker.internal`。
   - **k3d（Docker 里跑 k3s）**：`DB_HOST=host.k3d.internal`（k3d 提供的指向宿主机的主机名）。
   - **裸机 k3s**：k3s 无内置 host 别名，需用宿主机 IP，例如 `DB_HOST=192.168.1.100`（换成你机器的实际 IP）。
   - 其它如 `DB_USER`、`DB_PASSWORD`、`DB_NAME`（如 `twixter-store`）、`DB_PORT=5432` 按本机 Postgres 填写；本机若未开 SSL 可设 `DB_SSLMODE=disable`。
   - **本机 Postgres 必须接受来自 Minikube 的连接**：在 Postgres 的 `postgresql.conf` 里设 `listen_addresses = '*'`（或至少包含本机对 Minikube 网段的 IP），在 `pg_hba.conf` 里允许 Minikube 网段（如 `192.168.0.0/16` 或 `10.0.0.0/8`）的 TCP 连接，然后重启 Postgres。否则 Pod 仍会连不上。

   **如何查看并确认上述配置：**

   1. **找到配置文件路径**：用 `psql` 连上本机 Postgres 后执行 `SHOW config_file;` 得到 `postgresql.conf` 路径，执行 `SHOW hba_file;` 得到 `pg_hba.conf` 路径。常见位置：Windows 安装版多在 `C:\Program Files\PostgreSQL\<版本>\data\`；Linux 多在 `/etc/postgresql/<版本>/main/` 或 `/var/lib/pgsql/data/`。
   2. **查看 `listen_addresses`**：用编辑器打开 `postgresql.conf`，搜索 `listen_addresses`。确认其值为 `listen_addresses = '*'` 或至少包含本机对 Minikube 可见的 IP。若该行被 `#` 注释，需去掉注释并改值后保存。
   3. **查看 `pg_hba.conf`**：用编辑器打开 `pg_hba.conf`，查看 “TYPE DATABASE USER ADDRESS METHOD” 规则。需有一行允许 Minikube 网段的 TCP 连接，例如：`host    all    all    192.168.0.0/16    scram-sha-256` 或 `host    all    all    10.0.0.0/8    scram-sha-256`（网段按你 Minikube 实际网段调整）。若无则添加并保存。
   4. **修改后重启 Postgres**：Windows 在「服务」（即 Windows 的服务管理界面：按 `Win+R` 输入 `services.msc` 回车可打开，或开始菜单搜索「服务」）里找到 `postgresql-x64-<版本>`，右键 → 重新启动；Linux 使用 `sudo systemctl restart postgresql`（或 `postgresql-16` 等，视发行版而定）。

2. **Redis 连接失败（如 dial tcp: lookup redis-service: no such host 或 dial tcp [::1]:6379: connection refused）**  
   - 后端默认需要 Redis（`REDIS_ADDR`）。若未在 Secret 中设置，应用会使用默认值 `localhost:6379`，在 Pod 内 localhost 指向 Pod 自身，会报 `[::1]:6379: connection refused`。  
   - **方案 A（推荐，集群内 Redis）**：仓库已在 `backend/k8s/` 下提供 `redis-deployment.yaml` 和 `redis-service.yaml`，`kubectl apply -k k8s/` 会一并部署 Redis 并创建名为 `redis-service` 的 Service（端口 6379）。  
     1. 在 **`backend/k8s/.env`** 中设置：`REDIS_ADDR=redis-service:6379`、`REDIS_PASSWORD=`、`REDIS_DB=0`（若未设置 `REDIS_ADDR`，后端会退回到 `localhost:6379` 导致连不上）。  
     2. 在仓库根目录执行：`kubectl apply -k k8s/`（部署 Redis 并重新生成 Secret），再执行 `kubectl rollout restart deployment/twixter-backend`。  
     3. 确认 Secret 里已是新值（应看到 `redis-service:6379`；若为空或 `localhost:6379`，说明未用 `backend/k8s/.env` 或未执行 `kubectl apply -k k8s/`）：  
        - **Linux / macOS**：`kubectl get secret twixter-backend-secret -o jsonpath='{.data.REDIS_ADDR}' | base64 -d`  
        - **Windows（PowerShell）**：`[System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String((kubectl get secret twixter-backend-secret -o jsonpath='{.data.REDIS_ADDR}')))`  
     4. 确认：`kubectl get pods` 中 Redis Pod 为 Running，`kubectl logs deployment/twixter-backend -f` 无 Redis 连接错误。  
   - **方案 B**：使用外部 Redis（如宿主机上的 Redis），在 `backend/k8s/.env` 里设置 `REDIS_ADDR=host.minikube.internal:6379`（或你的 Redis 地址:端口），重新生成 Secret 并重启 backend。

2.1 **Kafka 未配置（Failed to initialize Kafka service: ... dial tcp [::1]:9092: connection refused）**  
   - 后端视频上传与处理依赖 Kafka。若未在 Secret 中设置 `KAFKA_BROKERS`，应用会使用默认值 `localhost:9092`，在 Pod 内会连不上。  
   - **方案 A（推荐，集群内 Kafka）**：仓库已在 `backend/k8s/` 下提供 `zookeeper-deployment.yaml`、`zookeeper-service.yaml`、`kafka-deployment.yaml`、`kafka-service.yaml`，`kubectl apply -k k8s/` 会一并部署 Zookeeper 与 Kafka，并创建名为 `kafka` 的 Service（端口 9092）。  
     1. 在 **`backend/k8s/.env`** 中设置：`KAFKA_BROKERS=kafka:9092`。  
     2. 在仓库根目录执行：`kubectl apply -k k8s/`（部署 Zookeeper/Kafka 并重新生成 Secret），再执行 `kubectl rollout restart deployment/twixter-backend`。  
     3. 确认：`kubectl get pods` 中 `zookeeper-*` 与 `kafka-*` 为 Running，`kubectl logs deployment/twixter-backend` 无 Kafka 连接错误。  
   - **方案 B**：使用外部 Kafka，在 `backend/k8s/.env` 里设置 `KAFKA_BROKERS=你的 Kafka 地址:9092`，重新生成 Secret 并重启 backend。未配置时视频处理不可用，内容只读接口仍可用（见上文 4）。

3. **Database migration failed**  
   - 多为 DB 连接或权限问题，先按 1 确保能连上库，再查日志中的具体错误。

4. **/api/content/* 返回 404（如 /api/content/viewing-token、/api/content/list）**  
   - **原因**：内容相关路由由 ContentController 注册，而 ContentController 仅在 ContentService 存在时注册。此前 ContentService 只有在 **S3 和 Kafka 都初始化成功** 时才会创建；若 `backend/k8s/.env` 未配置 AWS 或 Kafka，S3/Kafka 初始化失败 → ContentService 未创建 → 内容路由未注册 → 请求返回 404。  
   - **处理**：仓库已修改后端逻辑：S3 或 Kafka 初始化失败时改用占位实现（`backend/service/stub_services.go` 中的 `NoopS3Service` / `NoopKafkaService`），**始终创建 ContentService**，因此 `/api/content/list`、`/api/content/viewing-token`、`/api/content/:id` 等只读接口会正常注册；上传、流式播放、GIF 预览等依赖 S3/Kafka 的接口在未配置时会返回 503 或明确错误。  
   - **你需要做的**：重新构建后端镜像并重启 backend 部署（例如在 Minikube 内执行 `docker build -t damonleelcx/twixter.store-backend:latest ./backend`，再 `kubectl rollout restart deployment/twixter-backend`）。无需在 `.env` 中配置 S3/Kafka 即可使用内容列表与 viewing-token 等只读接口。

**5. 用端口转发临时验证（不依赖 Ingress）**

```bash
# 后端
kubectl port-forward svc/twixter-backend 8080:8080
# 另开终端访问 http://localhost:8080/health

# 前端
kubectl port-forward svc/twixter-frontend 3000:3000
# 另开终端访问 http://localhost:3000
```

**Windows（CMD / PowerShell）**：开两个终端。终端 1：`kubectl port-forward svc/twixter-backend 8080:8080`，浏览器访问 `http://localhost:8080/health`。终端 2：`kubectl port-forward svc/twixter-frontend 3000:3000`，浏览器访问 `http://localhost:3000`。

若 port-forward 能访问而通过域名不能，问题在 **hosts 或 Ingress**；若 port-forward 也不能访问，问题在 **Pod/应用**。

**5.1 port-forward 前端时出现 Connection refused / lost connection to pod**

若执行 `kubectl port-forward svc/twixter-frontend 3000:3000` 后访问 `http://localhost:3000` 报错：`error forwarding port 3000 to pod ... Connection refused` 或 `lost connection to pod`，而 `kubectl get pods -l app=twixter-frontend` 显示 Pod 为 `Running`、`READY 1/1`，说明端口转发已建立，但**容器内 3000 端口无人监听**。Next.js standalone 未设置 `HOSTNAME` 时，会按容器主机名（如 `twixter-frontend-xxx:3000`）监听，不监听 `127.0.0.1:3000`，kubectl port-forward 连的是容器内 127.0.0.1:3000，因此会被拒绝。

**处理**：仓库已在 `frontend/k8s/configmap.yaml` 中为前端设置 **`HOSTNAME=0.0.0.0`**，使 Next.js 监听所有接口。修改后需重新应用并重启前端：在仓库根目录执行 `kubectl apply -k k8s/`，再执行 `kubectl rollout restart deployment/twixter-frontend`；等新 Pod 就绪（`kubectl get pods -l app=twixter-frontend` 为 Running、1/1）后再试 port-forward 与 `http://localhost:3000`。

**5.2 前端能打开但 API 请求 api.twixter.local 报 ERR_CONNECTION_TIMED_OUT**

若通过 `http://localhost:3000`（port-forward 前端）能打开页面，但控制台出现 `GET http://api.twixter.local/api/auth/me net::ERR_CONNECTION_TIMED_OUT` 等，说明**前端页面在本机浏览器里运行，API 请求由浏览器发往 api.twixter.local**；若 api.twixter.local 不可达，就会超时。

**处理**：必须让 **api.twixter.local** 在本机可访问。

1. **确认 hosts**：本机 hosts 里必须有 `<MINIKUBE_IP>/127.0.0.1 api.twixter.local`（`minikube ip` 得到当前 IP）。Windows：`C:\Windows\System32\drivers\etc\hosts`。终端执行 `ping api.twixter.local` 应解析到 Minikube IP。
2. **确认 Minikube 隧道在跑**：以**管理员身份**开一个终端，执行 `minikube tunnel` 并**保持不关**。Ingress 使用 80 端口，浏览器访问 `http://api.twixter.local` 会连到 Minikube IP:80，隧道负责把 80 转到集群内后端。若隧道未开或 Windows 上 80 端口转发失败，api.twixter.local 会超时。
3. **先单独测后端**：在浏览器打开 `http://api.twixter.local/health`。若能看到健康检查结果，说明 api.twixter.local 已通，前端的 `/api/auth/me` 等请求也应能通；若 `/health` 也超时，问题在 hosts 或隧道。
4. **若隧道在 Windows 上 80 端口不可用（仅用 port-forward）**：前端构建时 API 写死为 `http://api.twixter.local`，浏览器会请求该域名（默认 80 端口）。要让 API 通，需让 api.twixter.local 在本机指向后端 port-forward 的端口：  
   - **hosts**：添加 `127.0.0.1 api.twixter.local`（Windows：`C:\Windows\System32\drivers\etc\hosts`，需管理员权限编辑）。  
   - **后端 port-forward**：`kubectl port-forward svc/twixter-backend 8080:8080`（保持运行）。  
   - **本地 API 代理**：在仓库根目录执行 `node scripts/local-api-proxy.js`。该脚本监听本机 80 端口并转发到 `http://127.0.0.1:8080`，浏览器访问 `http://api.twixter.local` 时会走 127.0.0.1:80 → 代理 → 127.0.0.1:8080。**Windows 上绑定 80 端口通常需以管理员身份**打开终端再运行上述命令。  
   - **若报错 `EADDRINUSE: address already in use 127.0.0.1:80`**：说明 80 端口已被占用（常见为 IIS、Skype、其他 Web 服务）。可先查占用进程：`netstat -ano | findstr :80`，记下最后一列 PID，再 `taskkill /PID <pid> /F` 结束该进程（或从「服务」里停止 IIS 等）；若不能释放 80，可改用其他端口：`set PORT=8081&& node scripts/local-api-proxy.js`，然后**重新构建前端**时传入 `NEXT_PUBLIC_API_URL=http://api.twixter.local:8081`，并重新部署前端，这样浏览器会请求 api.twixter.local:8081。  
   - 保持三个进程运行：① 后端 port-forward；② 本地 API 代理（本脚本）；③ 前端 port-forward（`kubectl port-forward svc/twixter-frontend 3000:3000`）。然后访问 `http://localhost:3000`，前端的 `/api/auth/me` 等请求会通过 api.twixter.local → 本机 80 → 代理 → 8080 正常返回。

**6. 若非 Minikube（如 Docker Desktop K8s、kind）**

- 需自行安装 Ingress Controller（如 nginx-ingress），并确认 `ingressClassName: nginx` 与集群中的 Ingress Class 一致。
- 访问方式可能为 `http://localhost` + 某端口，或需配置 hosts 指向 `127.0.0.1`，视具体环境而定。

---

## 在新 Linux 服务器上使用真实域名 twixter.store 部署

以下步骤适用于**从零**在一台新的 Linux 服务器上部署，并使用**真实域名 twixter.store**（API 使用 `api.twixter.store`，前端使用 `www.twixter.store` 或 `twixter.store`）。

### 前置条件

- 一台有**公网 IP** 的 Linux 服务器（如 Ubuntu 22.04 / Debian 12 / **Rocky Linux 8**，推荐至少 2 CPU、4GB 内存）
- 已购买并拥有域名 **twixter.store**，且能在域名服务商处添加 DNS 记录
- 本机已安装 `kubectl`（用于从本机或 CI 执行 `kubectl apply`；若只在服务器上操作，在服务器上安装即可）

### 步骤 1：准备服务器（SSH、系统、防火墙）

1. **SSH 登录**（将 `<SERVER_IP>` 换成服务器公网 IP）：
   ```bash
   ssh root@<SERVER_IP>
   # 或 ssh your_user@<SERVER_IP>
   ```

2. **更新系统并安装基础工具**（以 Rocky Linux 8 为例）：
   ```bash
   sudo dnf update -y
   sudo dnf install -y curl git
   ```

3. **开放防火墙端口**（Rocky Linux 8 使用 firewalld）：
   ```bash
   sudo firewall-cmd --permanent --add-service=ssh      # 22/tcp
   sudo firewall-cmd --permanent --add-service=http      # 80/tcp（Let's Encrypt 校验及重定向）
   sudo firewall-cmd --permanent --add-service=https     # 443/tcp
   sudo firewall-cmd --reload
   sudo firewall-cmd --list-all
   ```

### 步骤 2：配置 DNS（twixter.store）

在域名服务商控制台为 **twixter.store** 添加 A 记录，指向服务器公网 IP：

| 类型 | 主机记录 | 记录值 | 说明 |
|------|----------|--------|------|
| A    | `@`      | `<SERVER_IP>` | 根域名 twixter.store |
| A    | `www`    | `<SERVER_IP>` | www.twixter.store（前端） |
| A    | `api`    | `<SERVER_IP>` | api.twixter.store（后端 API） |

保存后等待 DNS 生效（几分钟到几小时不等）。可用 `dig api.twixter.store +short` 或 `nslookup api.twixter.store` 验证是否已解析到 `<SERVER_IP>`。

### 步骤 3：在服务器上安装 Kubernetes 与 Ingress

任选其一即可。

**方案 A：k3s（单节点推荐，轻量；Rocky Linux 8 推荐）**

```bash
# Rocky Linux 8 / 通用：安装 k3s
curl -sfL https://get.k3s.io | sh -
sudo systemctl enable k3s
sudo systemctl start k3s
# Rocky 8 若启用 firewalld，放行 k3s API 与节点通信
sudo firewall-cmd --permanent --add-port=6443/tcp --add-port=10250/tcp
sudo firewall-cmd --reload
# 本机使用 kubectl
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config
export KUBECONFIG=~/.kube/config
# 可选：每次登录生效，echo 'export KUBECONFIG=~/.kube/config' >> ~/.bashrc
kubectl get nodes
```

k3s 自带 Traefik；若希望与现有文档一致使用 **nginx Ingress**，可安装 nginx-ingress 并禁用 Traefik：

```bash
# 若已安装 k3s 且要改用 nginx ingress，先禁用 traefik（无 traefik 时 xargs -r 不执行，不报错）
kubectl get deploy -n kube-system -o name | grep traefik | xargs -r kubectl delete -n kube-system
# 安装 nginx ingress
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/baremetal/deploy.yaml
# 若 LoadBalancer 一直 Pending，改为 NodePort 或 HostNetwork，或见方案 B
```

**方案 B：Minikube（driver=none，直接跑在宿主机）**

- **Rocky Linux 8 建议用方案 A（k3s）**：minikube 的 `--extra-config` 不支持 `kubeadm.ignorePreflightErrors`，在 Rocky 8（内核 4.18、cgroups v1）上 kubeadm 会报 SystemVerification，无法通过参数跳过。k3s 不用 kubeadm，无此限制。
- **Ubuntu/Debian**：可用 `curl -fsSL https://get.docker.com | sh` 安装 Docker。
- **Rocky Linux 8 / RHEL/CentOS 系**（若仍选 Minikube）：`get.docker.com` 不支持，请用 dnf + Docker 官方源安装：

```bash
# Rocky Linux 8：安装 Docker（get.docker.com 不支持 rocky，用 dnf）
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo dnf install -y docker-ce docker-ce-cli containerd.io
sudo systemctl enable --now docker
sudo usermod -aG docker $USER
# driver=none 时 Kubernetes 需要 conntrack、crictl、containernetworking-plugins（在 root 的 PATH / /opt/cni/bin）
sudo dnf install -y conntrack-tools
# CNI 插件（minikube none driver 必需），装到 /opt/cni/bin
CNI_VER=v1.4.0
sudo mkdir -p /opt/cni/bin
curl -sSL "https://github.com/containernetworking/plugins/releases/download/${CNI_VER}/cni-plugins-linux-amd64-${CNI_VER}.tgz" -o /tmp/cni-plugins.tgz
sudo tar zxvf /tmp/cni-plugins.tgz -C /opt/cni/bin
rm -f /tmp/cni-plugins.tgz
# Rocky 8 默认源无 cri-tools，从 GitHub 安装 crictl 到 /usr/local/bin（root 可见）
curl -sSL https://github.com/kubernetes-sigs/cri-tools/releases/download/v1.30.0/crictl-v1.30.0-linux-amd64.tar.gz -o crictl.tar.gz
sudo tar zxvf crictl.tar.gz -C /usr/local/bin
rm -f crictl.tar.gz
# K8s 1.24+ 与 Docker 运行时需要 cri-dockerd（minikube none driver）
# 在固定目录解压并安装到 /usr/bin（Rocky 上 sudo 的 secure_path 常不含 /usr/local/bin，故用 /usr/bin）
CRIDOCKERD_VER=0.3.22
cd /tmp && rm -rf cri-dockerd-install && mkdir cri-dockerd-install && cd cri-dockerd-install
curl -sSL "https://github.com/Mirantis/cri-dockerd/releases/download/v${CRIDOCKERD_VER}/cri-dockerd-${CRIDOCKERD_VER}.amd64.tgz" -o cri-dockerd.tgz
tar zxvf cri-dockerd.tgz
sudo install -m 0755 "$(find /tmp/cri-dockerd-install -name 'cri-dockerd' -type f | head -1)" /usr/bin/cri-dockerd
rm -rf /tmp/cri-dockerd-install
sudo which cri-dockerd
curl -sSL https://raw.githubusercontent.com/Mirantis/cri-dockerd/master/packaging/systemd/cri-docker.service -o /tmp/cri-docker.service
curl -sSL https://raw.githubusercontent.com/Mirantis/cri-dockerd/master/packaging/systemd/cri-docker.socket -o /tmp/cri-docker.socket
# 装到 /usr/bin 以便 root（sudo 的 secure_path）能找到；service 默认即 /usr/bin/cri-dockerd，无需改
sudo mv /tmp/cri-docker.service /tmp/cri-docker.socket /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cri-docker.socket
# 确认 root 能找到 cri-dockerd 且 socket 已启（若仍报 NOT_FOUND_CRI_DOCKERD 见下）
sudo which cri-dockerd
sudo systemctl status cri-docker.socket
# Rocky 8 none driver 还需：/etc/cni/net.d 存在、firewalld 放行 8443/10250、cgroups v1 时跳过校验（见下）
sudo mkdir -p /etc/cni/net.d
sudo firewall-cmd --permanent --add-port=8443/tcp --add-port=10250/tcp
sudo firewall-cmd --reload
# 登出再登入后安装 minikube（在可写目录下载，避免在已删的 /tmp/cri-dockerd-install 里写文件失败）
cd ~
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
# 运行时填 docker；cri-dockerd 是让 Docker 支持 K8s 1.24+ 的后端，minikube 仍选 docker
minikube start --driver=none --container-runtime=docker
minikube addons enable ingress
```

**若仍报 `/etc/cni/net.d` 不存在**：执行 `sudo mkdir -p /etc/cni/net.d`。

**若仍报 kernel 4.18 不支持 / cgroups v1 / SystemVerification**：minikube 的 `--extra-config` **不支持** `kubeadm.ignorePreflightErrors`，无法通过参数跳过该校验。**Rocky 8 上建议改用方案 A（k3s）**；若坚持用 Minikube，需升级内核到 5.x 或启用 cgroups v2。

**若仍报 `NOT_FOUND_CRI_DOCKERD` 或 `which: no cri-dockerd`**：说明二进制未装进 root 的 PATH。请按下面**手动安装**（每步执行后看输出）：

```bash
# 1. 在固定目录下载并解压
cd /tmp && rm -rf cri-dockerd-install && mkdir cri-dockerd-install && cd cri-dockerd-install
curl -sSL "https://github.com/Mirantis/cri-dockerd/releases/download/v0.3.22/cri-dockerd-0.3.22.amd64.tgz" -o cri-dockerd.tgz
tar zxvf cri-dockerd.tgz
# 2. 看解压出的内容（记下二进制实际路径，下面用）
ls -laR
# 3. 把二进制装到 /usr/bin（Rocky 上 root 的 PATH 常不含 /usr/local/bin，故用 /usr/bin）
sudo install -m 0755 "$(find /tmp/cri-dockerd-install -name 'cri-dockerd' -type f | head -1)" /usr/bin/cri-dockerd
# 若上面 find 为空，则手动指定：sudo install -m 0755 /tmp/cri-dockerd-install/cri-dockerd/cri-dockerd /usr/bin/cri-dockerd
# 4. 确认 root 能找到
sudo which cri-dockerd
sudo /usr/bin/cri-dockerd --version
# 5. 装 systemd 服务并启动（若尚未装；service 默认即 /usr/bin/cri-dockerd，无需改）
sudo curl -sSL -o /etc/systemd/system/cri-docker.service https://raw.githubusercontent.com/Mirantis/cri-dockerd/master/packaging/systemd/cri-docker.service
sudo curl -sSL -o /etc/systemd/system/cri-docker.socket https://raw.githubusercontent.com/Mirantis/cri-dockerd/master/packaging/systemd/cri-docker.socket
sudo systemctl daemon-reload && sudo systemctl enable --now cri-docker.socket
# 6. 删掉旧集群并用 cri-dockerd 重新起
minikube delete
minikube start --driver=none --container-runtime=docker
minikube addons enable ingress
```

若为 **Ubuntu/Debian**，可仅执行：

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# 登出再登入后安装 minikube（同上）
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
minikube start --driver=none
minikube addons enable ingress
```

**验证 Ingress Controller**：

```bash
kubectl get pods -n ingress-nginx   # 方案 A 用 nginx 时
# 或
kubectl get pods -n kube-system    # k3s 默认或 minikube
```

### 步骤 4：克隆仓库并配置生产环境变量

在服务器（或本机）上：

```bash
git clone <你的仓库 URL> twixter.store
cd twixter.store
```

1. **后端 Secret（生产用）**  
   复制并编辑 `backend/k8s/.env`（可从 `backend/.env.example` 或 `backend/.env` 复制），填写**生产**数据库、Redis、Kafka、Stripe、PayPal、S3 等：

   ```bash
   cp backend/.env.example backend/k8s/.env
   nano backend/k8s/.env
   ```

   必须包含且改为**生产值**的示例：

   ```env
   DB_HOST=你的Postgres主机
   DB_USER=...
   DB_PASSWORD=...
   DB_NAME=twixter-store
   DB_PORT=5432
   DB_SSLMODE=require
   REDIS_ADDR=redis-service:6379
   KAFKA_BROKERS=kafka:9092
   STRIPE_SECRET_KEY=sk_live_...
   PAYPAL_CLIENT_SECRET=...
   # 以及 S3、CORS 等按需配置
   ```

2. **Ingress 使用真实域名**  
   将 Ingress 中的 `api.twixter.local` / `www.twixter.local` 改为真实域名：

   - 编辑 `backend/k8s/ingress.yaml`：把 `host: api.twixter.local` 改为 `host: api.twixter.store`；若启用 TLS，取消注释 `tls` 并设置 `secretName`（见步骤 5）。
   - 编辑 `frontend/k8s/ingress.yaml`：把 `host: www.twixter.local` 改为 `host: www.twixter.store`（或 `twixter.store`）；若启用 TLS，同上。

### 步骤 5：启用 HTTPS（Let's Encrypt + cert-manager，推荐）

1. **安装 cert-manager**：

   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml
   kubectl wait --for=condition=Available deployment --all -n cert-manager --timeout=120s
   ```

2. **创建 ClusterIssuer（Let's Encrypt）**：

   ```bash
   cat <<EOF | kubectl apply -f -
   apiVersion: cert-manager.io/v1
   kind: ClusterIssuer
   metadata:
     name: letsencrypt-prod
   spec:
     acme:
       server: https://acme-v02.api.letsencrypt.org/directory
       email: twixter.store@gmail.com
       privateKeySecretRef:
         name: letsencrypt-prod
       solvers:
         - http01:
             ingress:
               class: nginx
   EOF
   ```

   将 `your-email@example.com` 换成你的邮箱。

3. **在 Ingress 中启用 TLS**：  
   在 `backend/k8s/ingress.yaml` 和 `frontend/k8s/ingress.yaml` 中取消注释 `tls` 段，并改为使用 cert-manager 自动签发证书（推荐用 annotation，不用手写 secretName）：

   **backend/k8s/ingress.yaml** 示例：

   ```yaml
   metadata:
     annotations:
       cert-manager.io/cluster-issuer: "letsencrypt-prod"
   spec:
     rules:
       - host: api.twixter.store
         ...
     tls:
       - hosts:
           - api.twixter.store
         secretName: twixter-backend-tls
   ```

   **frontend/k8s/ingress.yaml** 同理，`host` 与 `tls.hosts` 改为 `www.twixter.store`（或 `twixter.store`），`secretName` 如 `twixter-frontend-tls`。  
   cert-manager 会自动创建上述 Secret 并续期。

4. **若不使用 cert-manager：手动创建 TLS Secret**  
   若使用自签名或已有证书，可在对应命名空间下手动创建 Secret（替换为你的证书与私钥文件）：

   ```bash
   kubectl create secret tls twixter-backend-tls --cert=api.twixter.store.crt --key=api.twixter.store.key
   kubectl create secret tls twixter-frontend-tls --cert=www.twixter.store.crt --key=www.twixter.store.key
   ```

   Ingress 中的 `tls.secretName` 已配置为 `twixter-backend-tls` / `twixter-frontend-tls`，创建上述 Secret 后 Ingress 即可使用 HTTPS。

### 步骤 6：前端构建时使用真实域名

前端镜像构建时需传入**生产环境**的 API 与站点地址（HTTPS）：

```bash
cd frontend
docker build -t damonleelcx/twixter.store-frontend:latest \
  --build-arg NEXT_PUBLIC_API_URL="https://api.twixter.store" \
  --build-arg NEXT_PUBLIC_APP_URL="https://www.twixter.store" \
  --build-arg NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY="pk_live_..." \
  --build-arg NEXT_PUBLIC_PAYPAL_CLIENT_ID="..." \
  --build-arg BACKEND_URL="http://twixter-backend:8080" \
  .
cd ..
```

**Rocky Linux 8 / k3s：从 `.env` 加载变量再构建**  
在 Rocky 8 上若使用 `frontend/k8s/.env` 存放 Stripe/PayPal 等 key，先加载再构建（与 Minikube 小节中的 bash 方式相同）。

**前置条件（避免出现 “Stripe is not configured”）**  
1. 必须在 **bash 或 Git Bash** 下执行下面命令（Windows 的 CMD 不支持 `set -a` 与 `. ./k8s/.env`，变量不会被加载）。  
2. 在 `frontend/k8s/.env` 中**必须**有一行：`NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_xxx` 或 `pk_test_xxx`，**等号两边不要有空格**，且不要用引号包住整行。  
3. 构建前可在同一 shell 里执行 `echo "STRIPE_KEY length: ${#NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY}"` 确认已加载（若为 0 说明未加载，需检查 .env 路径与格式）。

```bash
cd frontend
# 加载 k8s/.env，使 NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY 等传入下面的 docker build（仅 bash/Git Bash）
[ -f k8s/.env ] && set -a && . ./k8s/.env && set +a
docker build -t damonleelcx/twixter.store-frontend:latest \
  --build-arg NEXT_PUBLIC_API_URL="https://api.twixter.store" \
  --build-arg NEXT_PUBLIC_APP_URL="https://www.twixter.store" \
  --build-arg NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY="${NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY:-}" \
  --build-arg NEXT_PUBLIC_PAYPAL_CLIENT_ID="${NEXT_PUBLIC_PAYPAL_CLIENT_ID:-}" \
  --build-arg BACKEND_URL="http://twixter-backend:8080" \
  .
cd ..
```

- `set -a`：之后执行的变量赋值都会导出到环境；`. ./k8s/.env` 会执行 `frontend/k8s/.env` 里的 `KEY=VALUE`，从而把变量放进当前 shell 环境；`set +a` 关闭该行为。
- `"${NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY:-}"` 表示使用环境变量值，若未设置则为空；key 不要写进 README，放在 `frontend/k8s/.env` 即可。

- `NEXT_PUBLIC_API_URL` / `NEXT_PUBLIC_APP_URL` 必须使用 **https://api.twixter.store** 和 **https://www.twixter.store**，与 Ingress 及 DNS 一致。
- **BACKEND_URL**：构建时传入供 `next.config` 的 rewrite 使用；**运行时**在 K3s/K8s 中必须通过 ConfigMap（如 `frontend/k8s/configmap.yaml`）注入 `BACKEND_URL=http://twixter-backend:8080`，否则 SSR（如帖子详情页）请求后端会拿不到该变量（最终镜像的 runner 阶段不含此 ENV），可能回退到同源或 localhost 导致 ECONNREFUSED。
- 若镜像在 CI 或本机构建后推送到 Docker Hub，服务器上拉取同一镜像即可，无需在服务器上再构建。

### 步骤 7：构建后端镜像并部署

1. **后端镜像**（在项目根目录）：

   ```bash
   cd backend
   docker build -t damonleelcx/twixter.store-backend:latest .
   cd ..
   ```

   **环境变量（k3s / Rocky Linux 8）**  
   后端镜像**构建时**不需要加载 `.env`（镜像内不打包敏感配置）。  
   **运行时** env 由集群注入：在步骤 4 已编辑 `backend/k8s/.env` 并填写生产值；在仓库根目录执行 `kubectl apply -k k8s/` 时，Kustomize 会从 `backend/k8s/.env` 生成 Secret `twixter-backend-secret`，后端 Deployment 通过 `envFrom` 将该 Secret 与 ConfigMap 注入 Pod，因此无需在 `docker build` 时传 env。  
   若未做步骤 4，请先执行 `cp backend/.env.example backend/k8s/.env` 并编辑 `backend/k8s/.env`，再执行 `kubectl apply -k k8s/`。

2. **推送镜像到 Docker Hub**（若集群从 Hub 拉取）：

   ```bash
   docker push damonleelcx/twixter.store-backend:latest
   docker push damonleelcx/twixter.store-frontend:latest
   ```

   **本地构建且不推送到仓库时**（按集群类型二选一）：
   - **Minikube**：先 `eval $(minikube docker-env)` 再构建，则镜像在 Minikube 内可见，无需 push。
   - **k3s**：k3s 无 `minikube docker-env` 等价命令。可选其一：
     1. **导入到 k3s 的 containerd**：构建后执行  
        `docker save damonleelcx/twixter.store-backend:latest | sudo k3s ctr images import -`  
        `docker save damonleelcx/twixter.store-frontend:latest | sudo k3s ctr images import -`  
        则集群可直接使用该镜像（无需 push）。
     2. **安装 k3s 时使用本机 Docker**：安装时加 `--docker`（如  
        `curl -sfL https://get.k3s.io | sh -s - --docker`），则本机 `docker build` 的镜像与 k3s 共用同一 Docker，直接构建即可，无需 push。

   **k3s 从 Docker Hub 拉取镜像（ErrImagePull / ImagePullBackOff）**  
   k3s 使用自带的 containerd，**不会**使用本机 `docker login`。若从 Docker Hub 拉取镜像（含私有或需登录的仓库），需在集群内创建拉取凭据 Secret `regcred`（backend/frontend 的 Deployment 已配置 `imagePullSecrets: - name: regcred`）。在**已执行过 `docker login` 的机器**上执行其一即可：

   ```bash
   # 方式一：用本机 ~/.docker/config.json 生成（推荐，与 docker login 一致）
   kubectl create secret generic regcred \
     --from-file=.dockerconfigjson=$HOME/.docker/config.json \
     --type=kubernetes.io/dockerconfigjson \
     -n default
   ```

   ```bash
   # 方式二：直接填用户名/密码（替换为你的 Docker Hub 用户名与 Access Token 或密码）
   kubectl create secret docker-registry regcred \
     --docker-server=https://index.docker.io/v1/ \
     --docker-username=你的用户名 \
     --docker-password=你的密码或AccessToken \
     --docker-email=你的邮箱 \
     -n default
   ```

   若 Secret 已存在需更新：`kubectl delete secret regcred -n default` 再执行上面其一，然后 
   `kubectl rollout restart deployment/twixter-backend`
   `kubectl rollout restart deployment/twixter-frontend`。

3. **一次性部署**（在仓库根目录）：

   ```bash
   kubectl apply -k k8s/
   ```

4. **等待 Pod 就绪并检查 Ingress**：

   ```bash
   kubectl get pods
   kubectl get ingress
   kubectl get certificate -A
   ```

   若证书未就绪，可查看：`kubectl describe certificate twixter-backend-tls`（及 frontend 的），确认 ACME challenge 是否成功。

**Rocky Linux 8**  
在 Rocky Linux 8 上，步骤 5–7 中的 `kubectl`、`docker` 命令同样适用。若尚未安装 kubectl 或 Docker，可先执行：

- **安装 kubectl**：

  ```bash
  sudo dnf install -y curl
  curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
  sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
  ```

- **安装 Docker CE**：

  ```bash
  sudo dnf install -y dnf-plugins-core
  sudo dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
  sudo dnf install -y docker-ce docker-ce-cli containerd.io
  sudo systemctl enable --now docker
  sudo usermod -aG docker $USER
  ```

  注销并重新登录后，`docker` 无需 sudo。  
  若使用 **Podman** 替代 Docker（Rocky 8 默认可用）：`sudo dnf install -y podman`，步骤 6–7 中的 `docker build` / `docker push` 可改为 `podman build` / `podman push`，并视需要配置集群使用 Podman 拉取的镜像。

### 步骤 8：验证访问

- **后端**：`https://api.twixter.store/health`
- **前端**：`https://www.twixter.store`

若 80/443 由云厂商负载均衡或反向代理承接，需在该处将 80 转发到集群 NodePort 或 Ingress Controller 暴露的端口（k3s/minikube 单机常见为 NodePort 或 HostNetwork）。

**k3s（Rocky Linux）浏览器打不开 api.twixter.store / www.twixter.store 时**

1. **DNS / hosts**  
   - 生产：`api.twixter.store`、`www.twixter.store` 的 A 记录指向 **k3s 节点公网 IP**（如 msd6200 的 IP）。  
   - 本机调试：在**访问浏览器的电脑**上改 hosts，把上述域名指到 k3s 节点 IP（与 `kubectl get nodes -o wide` 里该节点 IP 或公网 IP 一致）。

2. **Ingress Controller**  
   当前 Ingress 使用 `ingressClassName: nginx`，k3s 默认是 **Traefik**。二选一：  
   - **用 nginx**：按步骤 5 安装 nginx ingress 并禁用 Traefik；  
   - **用 Traefik**：把 Ingress 的 `ingressClassName: nginx` 改为 `traefik`（或集群里实际 IngressClass 名），并去掉 nginx 专用 annotation，再 `kubectl apply`。

3. **80/443 暴露与防火墙**  
   - nginx ingress 若为 **NodePort**（如 80:31916、443:31379），本机 `curl http://127.0.0.1/` 会得到 000（80 端口无进程监听）。要让浏览器用 **http://api.twixter.store**（80 端口）访问，需让控制器直接占用节点 80/443：在 **k3s 节点**上执行：
     ```bash
     kubectl patch deploy ingress-nginx-controller -n ingress-nginx --type=merge -p '{"spec":{"template":{"spec":{"hostNetwork":true,"dnsPolicy":"ClusterFirstWithHostNet"}}}}'
     kubectl rollout status deploy/ingress-nginx-controller -n ingress-nginx
     ```
     之后控制器会监听节点 80/443；防火墙已放行 http/https 即可用 `http://api.twixter.store/health`、`http://www.twixter.store` 访问。  
     **若 rollout status 一直卡在 “1 old replicas are pending termination”**：新 Pod 可能 Pending，事件里常见 `didn't have free ports for the requested pod ports`，说明节点 80/443 已被占用（多为 **k3s 自带的 Traefik**）。  
     1）先**回滚**：`kubectl rollout undo deploy/ingress-nginx-controller -n ingress-nginx`，然后放行 NodePort 31916/31379，用 `http://api.twixter.store:31916/health`、`http://www.twixter.store:31916`。  
     2）若要坚持 hostNetwork 用 80/443：先释放节点 80/443。若 `sudo ss -tlnp | grep -E ':80|:443'` 显示 **httpd（Apache）** 在监听，可停用：`sudo systemctl stop httpd`（或 `sudo systemctl disable --now httpd`）。若为 **Traefik**：`kubectl get deploy -n kube-system -o name | grep traefik | xargs -r kubectl delete -n kube-system`。确认 80/443 无输出后，再执行上面的 `kubectl patch ... hostNetwork` 并 `rollout status`。  
     3）若需**保留 Apache** 且用 80/443 访问站点：不要给 nginx ingress 开 hostNetwork；保持 NodePort（如 31916/31379），在 Apache 里配置**反向代理**，把 `api.twixter.store`、`www.twixter.store` 的 80/443 代理到 `http://127.0.0.1:31916` 和 `https://127.0.0.1:31379`（或仅 HTTP 代理到 31916），详见下方「Apache 反向代理到 NodePort」。  
   - 若不想改控制器，可放行 NodePort 并用 `http://api.twixter.store:31916`、`https://www.twixter.store:31379`（需同时放行 31916、31379）。
   - **Apache 反向代理到 NodePort**（本机已用 httpd 占 80/443 时）：保持 nginx ingress 为 NodePort（80→31916，443→31379）。在 Apache 中为 `api.twixter.store`、`www.twixter.store` 建 VirtualHost，启用 `mod_proxy`、`mod_proxy_http`（HTTPS 时还有 `mod_ssl`、`mod_proxy_connect`），例如：
     ```apache
     # /etc/httpd/conf.d/twixter-proxy.conf（示例，路径以实际为准）
     <VirtualHost *:80>
       ServerName api.twixter.store
       ProxyPreserveHost On
       ProxyPass / http://127.0.0.1:31916/
       ProxyPassReverse / http://127.0.0.1:31916/
     </VirtualHost>
     <VirtualHost *:80>
       ServerName www.twixter.store
       ProxyPreserveHost On
       ProxyPass / http://127.0.0.1:31916/
       ProxyPassReverse / http://127.0.0.1:31916/
     </VirtualHost>
     ```
     然后 `sudo systemctl reload httpd`。访问 `http://api.twixter.store/health`、`http://www.twixter.store` 即经 Apache 转到 nginx ingress 的 NodePort。
   - 在 **k3s 节点**上放行 80/443（或对应 NodePort）：  
     `sudo firewall-cmd --permanent --add-service=http --add-service=https && sudo firewall-cmd --reload`

4. **前端 Pod 0/1 READY / 503 Service Temporarily Unavailable**  
   若 `kubectl get pods` 里 frontend 为 `0/1 Running` 或 Ingress 返回 **503**，说明 nginx 找不到就绪的前端 Pod。先查：`kubectl get pods -l app=twixter-frontend`、`kubectl get endpoints twixter-frontend`（无 addresses 即无 Ready Pod）。修前端：看日志 `kubectl logs deployment/twixter-frontend --tail=80`；若 OOM 则提高 `resources.limits.memory` 与探针 `initialDelaySeconds`，再 `kubectl apply -k frontend/k8s/` 并 `kubectl rollout restart deployment/twixter-frontend`。等 frontend 变为 1/1 Running 后再访问。

5. **帖子页 500 / ECONNREFUSED（SSR 连不上后端）**  
   若访问 `/en/post/10` 等出现 500 或 “fetch failed / ECONNREFUSED”，说明前端 Pod 内 **BACKEND_URL** 未注入或后端不可达。先确认：  
   - ConfigMap 已应用且含 `BACKEND_URL`：`kubectl get configmap twixter-frontend-config -o yaml`（应有 `BACKEND_URL: "http://twixter-backend:8080"`）。  
   - 前端 Pod 已注入该变量：`kubectl exec deployment/twixter-frontend -- env | grep BACKEND_URL`（应输出 `BACKEND_URL=http://twixter-backend:8080`）。  
   - 后端服务与 Pod 正常：`kubectl get svc twixter-backend`、`kubectl get pods -l app=twixter-backend`。  
   若 BACKEND_URL 为空，执行 `kubectl apply -f frontend/k8s/configmap.yaml`（或 `kubectl apply -k frontend/k8s/`），再 `kubectl rollout restart deployment/twixter-frontend`。

6. **“Your connection isn’t secure” / TLS 证书无效**  
   Ingress 若配置了 TLS 但未安装 **cert-manager** 或证书未签发，浏览器会报不安全。可暂时用 **http://** 访问（如 `http://www.twixter.store`）；若 nginx 强制跳 HTTPS，可在 Ingress 上增加注解 `nginx.ingress.kubernetes.io/ssl-redirect: "false"` 临时关闭跳转。长期方案：安装 cert-manager、创建 ClusterIssuer（如 letsencrypt-prod），并确保 80 开放供 ACME HTTP-01 校验，证书签发后即可正常用 https。

### 生产部署小结（twixter.store）

| 项目 | 本地/Minikube | 生产（twixter.store） |
|------|----------------|------------------------|
| 前端站点 | http://www.twixter.local | https://www.twixter.store |
| 后端 API | http://api.twixter.local | https://api.twixter.store |
| DNS | hosts 指向 Minikube IP | A 记录 @ / www / api → 服务器 IP |
| TLS | 可选 | cert-manager + Let's Encrypt |
| 前端构建参数 | NEXT_PUBLIC_API_URL=http://api.twixter.local | NEXT_PUBLIC_API_URL=https://api.twixter.store |
| Ingress host | api.twixter.local / www.twixter.local | api.twixter.store / www.twixter.store |

故障排查可参考上文「步骤 6：无法访问时的故障排查」；生产环境需额外确认数据库、Redis、Kafka、Stripe/PayPal 等均可从集群内访问，且 `backend/k8s/.env` 中为生产配置。

---

For Docker/Kubernetes and video processing details, see:

- `backend/README.Docker.md`
- `backend/README.Kubernetes.md`
- `frontend/README.Kubernetes.md` — frontend K8s (Next.js)
- `backend/README.VIDEO_PROCESSING.md`


### 原因

后端跑在 Pod 里。在 Pod 里，`localhost` / `127.0.0.1` 指的是 Pod 自己，不是你的 Windows 本机，所以连不上你本机的 Postgres。

### 1. 改 backend/k8s/.env 里的 DB 配置

在 `backend/k8s/.env` 里，把 `DB_HOST` 改成集群能解析到你本机的地址：

- **Minikube**：`DB_HOST=host.minikube.internal`
- **Docker Desktop 自带的 K8s**：`DB_HOST=host.docker.internal`
- **k3d（Docker 里跑 k3s）**：`DB_HOST=host.k3d.internal`
- **裸机 k3s**：无内置 host 别名，用宿主机 IP，例如 `DB_HOST=192.168.1.100`（换成你机器的实际 IP）

其它按你本机 Postgres 填写，例如：

```env
DB_HOST=host.minikube.internal
DB_USER=postgres
DB_PASSWORD=你的密码
DB_NAME=twixter-store
DB_PORT=5432
DB_SSLMODE=disable
```

不要再用 `localhost` 或 `127.0.0.1` 作为 `DB_HOST`。

### 2. 让本机 Postgres 接受来自 Minikube 的连接

Postgres 默认只监听 `127.0.0.1`，Minikube 的 Pod 在别的网段，需要：

- **postgresql.conf**：找到 `listen_addresses`，改成：
  ```conf
  listen_addresses = '*'
  ```
  或至少包含本机在 Minikube 网段上的 IP。

- **pg_hba.conf**：加一行，允许 Minikube 网段访问（示例）：
  ```conf
  host all all 192.168.0.0/16 scram-sha-256
  ```
  或 `host all all 10.0.0.0/8 scram-sha-256`（视 Minikube 实际网段而定）。

重启 Postgres 服务（Windows：在「服务」里找到 `postgresql-x64-<版本>` 右键重新启动；打开方式：`Win+R` → 输入 `services.msc` 回车，或开始菜单搜索「服务」）。

### 3. 重新部署并重启后端

在仓库根目录执行：

```bash
kubectl apply -k k8s/
kubectl rollout restart deployment/twixter-backend
```

然后看 Pod 是否正常：

```bash
kubectl get pods
kubectl logs deployment/twixter-backend
```

若仍报错，把新的 `kubectl logs` 最后几行贴出来即可。