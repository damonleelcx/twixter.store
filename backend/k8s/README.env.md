# backend/k8s 使用说明

## Secret 从哪个 .env 生成？

Kustomize 的 `secretGenerator` 读取的是 **本目录下的 `.env`**，即 **`backend/k8s/.env`**（不是 `backend/.env`）。

- 从**仓库根目录**执行 `kubectl apply -k k8s/` 时，会先应用根目录的 `k8s/`，再应用 `backend/k8s/`；生成 Secret 时使用的仍是 **`backend/k8s/.env`**。
- 若你只在 `backend/.env` 里改了 `REDIS_ADDR`，Pod 不会拿到新值，因为 Secret 来自 **`backend/k8s/.env`**。

## 正确步骤

1. **编辑或创建 `backend/k8s/.env`**  
   确保其中有（无空格、无引号）：
   ```env
   REDIS_ADDR=redis-service:6379
   REDIS_PASSWORD=
   REDIS_DB=0
   KAFKA_BROKERS=kafka:9092
   FRONTEND_URL=https://www.twixter.store
   SMTP_HOST=msd6200.mjhst.com
   SMTP_PORT=587
   SMTP_USER=support@twixter.store
   SMTP_PASSWORD=你的邮箱密码
   ```
   其中 `FRONTEND_URL` 用于密码重置邮件中的链接；`SMTP_*` 用于发送密码重置邮件（可选，未配置时仅打印 token 不发邮件）。若使用 465 端口隐式 TLS，可设置 `SMTP_PORT=465` 和 `SMTP_USE_TLS=1`。
   若使用集群内 Kafka（`kubectl apply -k k8s/` 会部署 Zookeeper 与 Kafka），必须设置 `KAFKA_BROKERS=kafka:9092`，否则后端会使用默认 `localhost:9092` 导致连接失败。
   若你平时只维护 `backend/.env`，可先复制一份到本目录再改：
   ```bash
   cp backend/.env backend/k8s/.env
   ```
   然后在本目录的 `.env` 里加上/改成上述三行。

2. **在仓库根目录重新应用并重启后端**
   ```bash
   kubectl apply -k k8s/
   kubectl rollout restart deployment/twixter-backend
   ```

3. **确认 Secret 里已是新值**
   ```bash
   kubectl get secret twixter-backend-secret -o jsonpath='{.data.REDIS_ADDR}' | base64 -d
   ```
   应输出：`redis-service:6379`。若为空或是 `localhost:6379`，说明用的不是 `backend/k8s/.env` 或未执行 `kubectl apply -k k8s/`。

4. **确认 Pod 已重启并看日志**
   ```bash
   kubectl get pods -l app=twixter-backend
   kubectl logs deployment/twixter-backend -f
   ```
   Pod 的启动时间应在你执行 `rollout restart` 之后，且不应再出现 `[::1]:6379` 或 `localhost:6379`。

## 数据库在 k3s 节点本机（裸机 k3s）

后端跑在 Pod 里时，Pod 内的 `localhost` 指向 Pod 自己，**不是**宿主机。若 Postgres 装在 **k3s 所在的那台机器** 上：

- 建议用 **Downward API 把 `DB_HOST` 设为节点 IP**（`status.hostIP`），这样 Pod 连宿主机不会遇到 NAT/回环问题；做法见下方「数据库 `connection refused`」里的 **数据库在本机** 小节。
- 宿主机上的 Postgres 须监听 5432：`listen_addresses = '*'` 或包含该节点 IP，不能只写 `localhost`，否则 Pod 连不上。
- 若改用**托管 Postgres**（如 RDS），在 `deployment.yaml` 里不要加 `DB_HOST` 的 `fieldRef`，改由 Secret（即 `backend/k8s/.env`）提供 `DB_HOST`。

## 数据库 `connection refused`（K3s 访问外部 Postgres）

若后端日志出现 `dial tcp 208.x.x.x:5432: connect: connection refused`，说明 **TCP 层面无法连到目标地址**（未建连即被拒绝或中间设备 RST）。Postgres 已在主机上监听 `0.0.0.0:5432` 时，重点排查 **从 Pod 到主机的网络与防火墙**。

### 重启后能连、一请求（如 fetch content）就 refused

若 **重启后端后能连上 DB，但一发起业务请求（如获取内容）又报 connection refused**，常见原因：

1. **空闲连接被防火墙/中间设备关闭后，连接池复用了已断开的连接**  
   应用已把 **ConnMaxIdleTime 默认改为 30 秒**（早于常见防火墙空闲超时），减少复用死连接。若仍出现，可在 Secret/ConfigMap 里设 `DB_CONN_MAX_IDLE_TIME=20`（秒）进一步缩短，并重新部署。
2. **Postgres 与 K8s 在同一台机，用公网 IP 连本机存在 NAT/回环**  
   若 **status.hostIP 就是公网 IP**（如 208.x.x.x），用 hostIP 无效。改用 **节点在 Pod 网段的 IP**（k3s 单节点多为 **10.42.0.1**）作为 `DB_HOST`，在 ConfigMap 中设置（见下方「数据库在本机」）。

### 从 Pod 内验证连通性

在集群里起一个临时 Pod，用同一 `DB_HOST` 测试能否连到 5432：

```bash
# 用与后端相同的 namespace（如 default）
kubectl run -it --rm debug-net --image=alpine --restart=Never -- sh
# 在 Pod 内执行（把 208.122.213.192 换成你的 DB_HOST）：
apk add --no-cache postgresql-client
nc -zv 208.122.213.192 5432
# 或直接试 psql：
psql "host=208.122.213.192 port=5432 user=twixter_store_user dbname=twixter-store connect_timeout=5" -c "select 1"
```

- 若 Pod 内 `nc`/`psql` 也报 **connection refused**：问题在 **网络/防火墙**（见下），不是应用代码。
- 若 Pod 内能连、仅后端连不上：再查后端环境变量（Secret 里的 `DB_HOST`/`DB_USER`/`DB_NAME` 等）和 pg_hba.conf 是否放行 Pod 网段。

### 在 DB 主机（208.122.213.192）上必查项

1. **防火墙放行 5432 入站（来源为 Pod/节点网段）**  
   - **firewalld**：  
     `sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="10.42.0.0/16" port port="5432" protocol="tcp" accept'`  
     （k3s 默认 Pod 网段为 `10.42.0.0/16`；若不同请改成你的 Pod CIDR。）  
     `sudo firewall-cmd --reload`  
   - **iptables**：放行来自 Pod 网段的 5432：  
     `sudo iptables -I INPUT -p tcp -s 10.42.0.0/16 --dport 5432 -j ACCEPT`  
   - 云主机：在安全组/ACL 中放行 5432，来源为 K8s 节点 IP 或 Pod 出网使用的 CIDR。

2. **PostgreSQL 监听**  
   已确认 `listen_addresses = '*'` 且 `netstat -lnpt` 显示 `0.0.0.0:5432` 即可。

3. **pg_hba.conf**  
   在 TCP 能连上之后，若出现认证错误，再在 DB 主机上把 Pod 网段加入 `pg_hba.conf` 并 `pg_ctl reload`，例如：  
   `host  all  all  10.42.0.0/16  scram-sha-256`

### 数据库在本机（Postgres 与 K8s 在同一台机）

若 Postgres 就装在 K8s 节点本机，Pod 连「本机公网 IP」（如 208.122.213.192）常因 **NAT/回环** 导致 connection refused 或 i/o timeout；且 **status.hostIP 在该机上往往就是公网 IP**，用 Downward API 也绕不开。

应改为用 **节点在 Pod 网段上的 IP** 作为 `DB_HOST`：k3s 单节点上 Pod 的默认网关多为 **10.42.0.1**，即节点在 Pod 网段的地址，Postgres 监听 `0.0.0.0:5432` 时会在该接口上接受连接。

1. **在 ConfigMap 里设置 `DB_HOST=10.42.0.1`**（ConfigMap 在 envFrom 中排在 Secret 之后，会覆盖 Secret 的 `DB_HOST`）。  
   本仓库 `configmap.yaml` 已默认包含 `DB_HOST: "10.42.0.1"`；若你的 Pod 网段网关不是 10.42.0.1，改成实际网关地址即可。  
2. 本机防火墙仍需放行 5432，来源为 Pod CIDR（如 `10.42.0.0/16`）。  
3. 若改用**托管 Postgres**（如 RDS），从 ConfigMap 中**删掉** `DB_HOST` 这一项，改由 Secret（`.env`）提供 `DB_HOST`。

应用并重启：`kubectl apply -k k8s/`（或 `kubectl apply -f backend/k8s/configmap.yaml`）后执行 `kubectl rollout restart deployment/twixter-backend`。  
验证：`kubectl exec deployment/twixter-backend -- env | grep DB_HOST` 应显示 `10.42.0.1`（或你设的网关）。

### 其他

- **连接数**：后端默认每实例最多 25 个连接（可设 `DB_MAX_OPEN_CONNS=10` 等）。多副本时总连接数不要超过 Postgres `max_connections`。
- **应用侧已做**：`/health` 重试、DSN `connect_timeout=5`、连接池 `ConnMaxLifetime`/`ConnMaxIdleTime`，减少断连复用导致的 connection refused。
