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

## 故障排查：CrashLoopBackOff（Backend / Frontend Pod 反复重启）

先看**具体报错**，再对症处理。

### 查看 Backend 日志（数据库 / Redis / Kafka 等初始化失败会直接退出）

```bash
kubectl logs deployment/twixter-backend --tail=100
# 若有多副本，指定 Pod 名
kubectl logs twixter-backend-65c9dfbbb7-tztdc --tail=100
```

常见原因：**数据库连不上**（见下方「no route to host」）、**Redis 连不上**（检查 `REDIS_ADDR` 是否指向集群内 `redis-service:6379`）、**Kafka 连不上**（检查 `KAFKA_BROKERS`）、或 **S3/Stripe 等配置缺失**（应用可能仍能启动，仅部分功能不可用）。

### 查看 Frontend 日志（Next.js 进程退出或 OOM）

```bash
kubectl logs deployment/twixter-frontend --tail=100
```

常见原因：**内存不足**（Next.js 在 128Mi/512Mi 下可能 OOM，可适当提高 limits）、**Secret 缺失**（若用 Kustomize `secretGenerator` 且 `frontend/k8s/.env` 不存在，Secret 可能为空，导致运行时缺变量；可在 `frontend/k8s/` 下建 `.env` 或改用手写 `secret.yaml`）。

### 查看 Pod 状态与退出原因

```bash
kubectl describe pod -l app=twixter-backend
kubectl describe pod -l app=twixter-frontend
```

关注 **Last State: Terminated** 的 **Exit Code** 和 **Reason**（例如 `OOMKilled` 表示内存超限，需提高 `resources.limits.memory`）。

### 建议

- **Backend**：先保证 DB、Redis 在集群内或可从 Pod 访问，再根据日志逐项修配置或网络。
- **Frontend**：若为 OOM，在 `frontend/k8s/deployment.yaml` 中适当提高 `resources.requests.memory` / `resources.limits.memory`（例如 256Mi / 1Gi）；若缺 Secret，确保 `frontend/k8s/.env` 存在或改用 `secret.yaml`。

## 故障排查：数据库连接 "no route to host"（k3s / Rocky Linux 8）

若出现类似错误：

```text
failed to connect to ... 208.122.213.192:5432: dial tcp ... connect: no route to host
```

说明从 **k3s 节点或 Pod** 无法路由到数据库主机（防火墙或网络未放行 5432）。按下面步骤排查。

### 1. 在 k3s 节点上测试能否连到数据库

在 **运行 k3s 的 Rocky Linux 主机** 上执行（将 `208.122.213.192` 换成你的 `DB_HOST`）：

```bash
nc -zv 208.122.213.192 5432
# 或
timeout 3 bash -c 'cat < /dev/null > /dev/tcp/208.122.213.192/5432' && echo OK || echo FAIL
```

- 若这里就失败（no route to host / connection refused / timeout），问题在 **节点到数据库** 之间的网络或防火墙，不是 Pod 配置。
- 若这里成功，再在 Pod 里测（见下）。

### 2. 在 Pod 内测试（确认 Pod 网络能出集群）

```bash
kubectl run -it --rm debug --image=busybox --restart=Never -- sh -c "nc -zv 208.122.213.192 5432 || true"
```

将 `208.122.213.192` 换成你的 `DB_HOST`。若 Pod 内也报 "no route to host"，多半是 **数据库侧或中间防火墙** 只允许了节点 IP，没有允许 k3s 的 Pod 网段。

### 3. 在数据库主机上放行 5432（数据库在自建 Linux 上时）

若 PostgreSQL 在 **你自己的一台 Linux 上**（例如 208.122.213.192），需要在该机上放行 5432。

**方式 A：firewalld（Rocky/RHEL 8 常用）**

**获取 k3s 节点实际 IP**：在 **运行 k3s 的那台 Rocky Linux** 上执行任一命令即可得到本机 IP，用于上面防火墙规则里的 `<K3S_NODE_IP>`：

```bash
# 方式 1：kubectl 显示的节点地址（通常是内网 IP）
kubectl get nodes -o wide

# 方式 2：本机主网卡 IP（示例为 eth0，按你机器网卡名改）
ip -4 addr show eth0 | grep -oP '(?<=inet\s)\d+(\.\d+){3}'

# 方式 3：简单列出所有 IPv4
hostname -I | awk '{print $1}'
```

若 k3s 和数据库不在同一网段（例如跨公网），需要放行的是 k3s 节点的 **出口/公网 IP**；可在 k3s 节点上执行 `curl -s ifconfig.me` 查看。

在 **数据库所在主机** 上执行，允许 k3s 节点 IP 访问 5432（把 `<K3S_NODE_IP>` 换成上面得到的实际 IP，或改用 k3s 的 Pod 网段如 `10.42.0.0/16`）：

```bash
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="<K3S_NODE_IP>" port port="5432" protocol="tcp" accept'
sudo firewall-cmd --reload
```

或临时开放 5432（仅测试用）：

```bash
sudo firewall-cmd --add-port=5432/tcp
```

并确认 PostgreSQL 监听外网接口（`postgresql.conf` 中 `listen_addresses = '*'` 或包含该网卡），以及 `pg_hba.conf` 允许对应用户从 k3s 节点/Pod 网段连接（见下方「PostgreSQL listen_addresses 与 pg_hba.conf」）。

**方式 B：iptables**

若未用 firewalld，可在数据库主机上用 iptables 放行 5432（示例：允许来自 10.42.0.0/16）：

```bash
sudo iptables -I INPUT -p tcp -s 10.42.0.0/16 --dport 5432 -j ACCEPT
```

**PostgreSQL listen_addresses 与 pg_hba.conf（Rocky Linux 8）**

在 **安装 PostgreSQL 的那台 Rocky Linux 8** 上操作。

1. **找到配置与数据目录**（常见路径）：

   ```bash
   sudo -u postgres psql -t -c "SHOW config_file;"
   sudo -u postgres psql -t -c "SHOW hba_file;"
   ```

   常见位置：`/var/lib/pgsql/data/postgresql.conf`、`/var/lib/pgsql/data/pg_hba.conf`。

2. **修改 listen_addresses**（允许接受非本机连接）：

   ```bash
   sudo sed -i "s/^#*listen_addresses.*/listen_addresses = '*'/" /var/lib/pgsql/data/postgresql.conf
   ```

   或手动编辑：

   ```bash
   sudo vi /var/lib/pgsql/data/postgresql.conf
   ```

   找到 `listen_addresses`，改为：

   ```ini
   listen_addresses = '*'
   ```

   （若只监听某网卡，可写该 IP，如 `listen_addresses = '192.168.1.10'`。）

3. **修改 pg_hba.conf**（允许 k3s 节点或 Pod 网段用密码连接）：

   ```bash
   sudo vi /var/lib/pgsql/data/pg_hba.conf
   ```

   在文件末尾增加一行（按需选一种）：

   - 允许 **单个 k3s 节点 IP**（将 `<K3S_NODE_IP>` 换成实际 IP）：

     ```text
     host    all    all    <K3S_NODE_IP>/32    scram-sha-256
     ```

   - 或允许 **k3s 默认 Pod 网段**（推荐，节点与 Pod 都能连）：

     ```text
     host    all    all    10.42.0.0/16    scram-sha-256
     ```

   - 或允许 **整段内网**（仅内网环境，示例 192.168.0.0/16）：

     ```text
     host    all    all    192.168.0.0/16    scram-sha-256
     ```

   若你的 Postgres 用的是 `md5` 认证，把上面的 `scram-sha-256` 改成 `md5`。

4. **重启 PostgreSQL 使配置生效**：

   ```bash
   sudo systemctl restart postgresql
   # 或
   sudo systemctl restart postgresql-15
   ```

   服务名可能带版本号，可用 `systemctl list-units 'postgresql*'` 查看。

5. **确认监听**：

   ```bash
   sudo ss -tlnp | grep 5432
   ```

   应看到 `0.0.0.0:5432` 或 `*:5432`，表示已监听所有接口。

**节点能连、Pod 不能连（no route to host）**

若在 **k3s 节点上** `nc -zv <DB_HOST> 5432` 成功，但 **Backend Pod** 仍报 `no route to host`，多半是 Pod 出网时用的源 IP 是 10.42.x.x，数据库或中间网络没有回包到 10.42 网段的路由。应让 Pod 访问外网时做 **SNAT（MASQUERADE）**，使数据库看到的是 **k3s 节点 IP**。

- **K3S_NODE_IP**：指 **运行 k3s 的那台机**（例如 msd6200）的 IP，不是数据库那台机的 IP。在数据库主机防火墙里“放行 K3S_NODE_IP”即放行来自 k3s 节点的连接。
- 在 **k3s 节点**上确认对 208.122.213.192 的出站做了 NAT（一般 k3s 会做，若未做可手动加）：
  ```bash
  sudo iptables -t nat -C POSTROUTING -d 208.122.213.192 -j MASQUERADE 2>/dev/null || sudo iptables -t nat -A POSTROUTING -d 208.122.213.192 -j MASQUERADE
  ```
  这样从 Pod 访问 208.122.213.192 时，源 IP 会变成节点 IP，数据库回包能回到节点再转给 Pod。
- 再在 **Pod 内**测一次：
  ```bash
  kubectl run -it --rm debug --image=busybox --restart=Never -- nc -zv 208.122.213.192 5432
  ```
  若成功，重启 Backend：`kubectl rollout restart deployment/twixter-backend`。

### 4. 云上托管数据库（RDS / Cloud SQL 等）

若 `DB_HOST` 是云厂商的地址（如 RDS），需要在控制台里把 **安全组 / 防火墙** 配置为允许 **k3s 节点的出口 IP**（或你用来访问外网的那个公网 IP）访问 5432。  
“no route to host” 有时是云侧直接丢弃未授权 IP 的包导致的。

### 5. 确认 k3s 节点本身出站未被拦

在 k3s 节点上：

```bash
sudo firewall-cmd --list-all
```

确认没有规则阻止到 `DB_HOST:5432` 的出站；Rocky 8 默认一般允许出站。

按上述顺序：先节点、再 Pod，再在数据库主机或云控制台放行 5432，通常即可解决 "no route to host"。

---

## 文件说明

| 文件 | 说明 |
|------|------|
| `secret.yaml` | 敏感配置模板（可选）：与 .env 键一致时可手填；用 .env 时由 Kustomize `secretGenerator` 从 `.env` 生成 Secret |
| `configmap.yaml` | 非敏感配置：TEMP_DIR、FFmpeg 路径等 |
| `deployment.yaml` | Backend 部署，从 Secret/ConfigMap 注入环境变量 |
| `service.yaml` | ClusterIP Service，端口 8080 |
| `ingress.yaml` | Ingress，配合 minikube ingress addon，主机名 `api.twixter.local` |

Postgres 使用托管实例时，不在此仓库中部署 Postgres Pod，仅通过 Secret 中的 `DB_*` 提供连接信息即可。
