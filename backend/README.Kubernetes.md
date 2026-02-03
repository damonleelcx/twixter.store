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

## 故障排查：Stripe API "no route to host"

若出现类似错误：

```text
Post "https://api.stripe.com/v1/checkout/sessions": dial tcp 34.x.x.x:443: connect: no route to host
```

说明从 **k3s 节点或 Pod** 无法访问 Stripe API（防火墙或网络未放行出站 443）。按下面顺序排查。

### 1. 先确认是节点还是 Pod 无路由

在 **k3s 节点** 上执行：

```bash
nc -zv api.stripe.com 443
```

再在 **Pod 内**（与 backend 同网络）执行：

```bash
kubectl run -it --rm debug --image=busybox --restart=Never -- sh -c "nc -zv api.stripe.com 443 || true"
```

- 若**节点上就失败**：说明节点本身出网 443 被拦（防火墙/安全组/无默认网关等），需在节点或上游防火墙上放行出站 443，或改用代理（见下）。
- 若**节点成功、Pod 失败**：多为 Pod 出网时源 IP 是 10.42.x.x，对端或中间设备未回包到该网段；可尝试在节点上对 Stripe 出口做 MASQUERADE（思路同数据库「节点能连、Pod 不能连」），或让 Pod 经代理出网。

### 方式 A：从网络侧解决——放行出站 443、检查路由/网关

若上面在节点或 Pod 内 `nc -zv api.stripe.com 443` 失败，按下面顺序在**网络侧**排查并修复。

#### 1. 检查 k3s 节点是否有默认网关、能否出网

在 **k3s 节点** 上执行：

```bash
# 是否有默认路由
ip route show default

# 能否 ping 通外网（若 ping 被禁，可跳过）
ping -c 2 8.8.8.8

# 本机出口公网 IP（若为空说明可能没出网或 NAT 异常）
curl -s --connect-timeout 3 ifconfig.me || true
```

- **没有 default 或下一跳不可达**：在节点或上游路由器上配好默认网关（如 `ip route add default via <网关IP>`），或检查网线/ DHCP。
- **有 default 但 ping/curl 不通**：多半是上游防火墙或运营商未放行出站，需在**上游设备**放行（见下）。

#### 2. 在 k3s 节点上放行出站 443（本机防火墙）

多数发行版**默认允许本机出站**，若你曾改过规则，需确保未拦截 443。

**firewalld（Rocky/RHEL/CentOS/Fedora）：**

```bash
# 查看默认区域（出站默认允许，一般无需额外规则）
sudo firewall-cmd --list-all
```

默认策略下**出站（output）是允许的**，通常不需要为 443 单独加规则。若你曾自定义过 `target: DROP` 或出站策略，再在对应区域放行出站 443（具体语法视版本而定，可查 `firewall-cmd --permanent --add-rich-rule --help`）。

**iptables：**

```bash
# 查看 OUTPUT 链是否 DROP 了 443
sudo iptables -L OUTPUT -n -v

# 若需放行出站 443（按你现有规则插入）
sudo iptables -A OUTPUT -p tcp --dport 443 -j ACCEPT
```

#### 3. 上游防火墙 / 路由器

若 k3s 节点在**内网**（如 192.168.x.x），出口经过**路由器或公司防火墙**：

- 在路由器/防火墙上允许 **k3s 节点 IP**（或整个内网网段）**出站访问 443**。
- 若使用 NAT，确保有**出站 NAT**，且没有「仅允许部分目的端口」的策略把 443 排除。

#### 4. 云上安全组 / ACL

若 k3s 跑在**云主机**（AWS / 阿里云 / GCP 等）：

- 在对应实例的**安全组 / 网络 ACL** 中，放行**出站（Egress）**：协议 TCP，目的端口 **443**，目的地址 `0.0.0.0/0` 或至少允许访问公网。

#### 5. 再次在节点与 Pod 内测试

```bash
# 节点上
nc -zv api.stripe.com 443

# Pod 内
kubectl run -it --rm debug --image=busybox --restart=Never -- sh -c "nc -zv api.stripe.com 443 || true"
```

若节点上仍报 **no route to host**：通常是**没有到目标网段的路由**（缺 default 或上游未回包），重点检查 1 和 3。  
若**节点成功、Pod 失败**：多为 Pod 出网未做 NAT（上游不认 10.42.x.x 源 IP），按下面「节点能连、Pod 不能连（Stripe）」处理。

#### 6. 节点能连、Pod 不能连（Stripe）

节点上 `nc -zv api.stripe.com 443` 成功，但 Pod 内同样命令报 **no route to host**，说明从 Pod 出去的包源 IP 是 10.42.x.x，上游或对端未回包到该网段。需在 k3s 节点上对 **Pod 出网** 做 MASQUERADE，使外网看到的是节点 IP。

**推荐：对「来自 Pod 网段、目的非集群」的流量统一做 MASQUERADE**（k3s 默认 Pod 网段为 10.42.0.0/16）：

```bash
# 在 k3s 节点上执行（持久化需视系统而定，见下）
sudo iptables -t nat -A POSTROUTING -s 10.42.0.0/16 ! -d 10.42.0.0/16 -j MASQUERADE
```

然后再次在 Pod 内测试：

```bash
kubectl run -it --rm debug --image=busybox --restart=Never -- sh -c "nc -zv api.stripe.com 443 || true"
```

若成功，重启 Backend：`kubectl rollout restart deployment/twixter-backend`。

**持久化**：上述 `iptables` 规则重启后可能丢失。持久化方式之一（Rocky/RHEL）：

```bash
# 安装 iptables-services（若未装），并保存当前规则
sudo yum install -y iptables-services
sudo iptables-save | sudo tee /etc/sysconfig/iptables
sudo systemctl enable iptables
```

或使用 firewalld 的 direct 规则（需在 MASQUERADE 生效后执行）：

```bash
sudo firewall-cmd --permanent --direct --add-rule ipv4 nat POSTROUTING 0 -s 10.42.0.0/16 ! -d 10.42.0.0/16 -j MASQUERADE
sudo firewall-cmd --reload
```

**若 MASQUERADE 后 Pod 仍报 no route to host**：可能是 k3s 与 firewalld/iptables 规则顺序、或节点未对 Pod 流量做转发。可先做下面诊断，再考虑用「方式 B：代理」兜底。

1. **确认节点已开启 IP 转发**：`cat /proc/sys/net/ipv4/ip_forward` 应为 `1`。若为 0：`echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward`，并持久化（如 `net.ipv4.ip_forward=1` 写入 `/etc/sysctl.d/99-forward.conf`）。
2. **看 NAT 规则是否生效**：`sudo iptables -t nat -L POSTROUTING -n -v --line-numbers`，确认含 `10.42.0.0/16` 的 MASQUERADE 规则存在且有包计数（若有）。
3. **试 firewalld 的 masquerade**（部分环境以 firewalld 为主）：  
   `sudo firewall-cmd --permanent --add-masquerade`，`sudo firewall-cmd --reload`，再在 Pod 内测 `nc -zv api.stripe.com 443`。
4. **兜底：在节点本机跑 HTTP 代理**（节点能连 Stripe，Pod 连节点即可）：在 k3s **节点**上装并启动 TinyProxy（监听 0.0.0.0:8888），放行 Pod 网段 `10.42.0.0/16`；Backend 的 ConfigMap/Secret 里设 `HTTPS_PROXY=http://<节点IP>:8888`（如 `http://208.122.213.192:8888`）。Pod 访问的是节点 IP，不经过出网 NAT，即可绕过「Pod 出网 no route」问题。详见下方「方式 B」与「在节点上跑代理（Pod 出网 NAT 仍不通时）」。
5. **持久化**：iptables 规则持久化见上。

### 方式 B：通过 HTTP 代理出网（推荐：节点无法直连外网时）

若集群只能经 HTTP/HTTPS 代理访问外网，让 Backend 通过代理访问 Stripe 即可。

1. **设置代理环境变量**（二选一）：
   - **ConfigMap**：编辑 `backend/k8s/configmap.yaml`，将 `HTTPS_PROXY` / `HTTP_PROXY` 改为你的代理地址，例如：
     ```yaml
     HTTPS_PROXY: "http://your-proxy-host:3128"
     HTTP_PROXY: "http://your-proxy-host:3128"
     ```
   - **Secret**（或 .env）：在 `backend/k8s/secret.yaml` 的 `stringData` 中增加（若用 Kustomize 从 `.env` 生成，则在 `backend/k8s/.env` 中增加）：
     ```yaml
     HTTPS_PROXY: "http://your-proxy-host:3128"
     ```
2. **确保 Pod 能访问代理**：在 Pod 内测试 `nc -zv your-proxy-host 3128`，若不通需先解决到代理的网络或防火墙。
3. **应用并重启**：
   ```bash
   kubectl apply -f backend/k8s/configmap.yaml
   # 若改的是 secret，则 apply secret 或 kubectl apply -k backend/k8s/
   kubectl rollout restart deployment/twixter-backend -n default
   ```

Backend 启动 Stripe 服务时会读取 `HTTPS_PROXY`/`HTTP_PROXY`，Stripe 请求会经代理发出，从而避免直连时的 "no route to host"。

### 什么是「代理主机」？如何搭建？

**代理主机（proxy host）** 是一台提供 **HTTP/HTTPS 代理** 的服务器：你的 Backend 不直接连 Stripe，而是把请求发给代理，由代理代替你去连 `api.stripe.com`，再把响应回给你。这样只要「Backend → 代理」和「代理 → 互联网」两条路通，Stripe 就能用。

- **何时需要**：k3s 节点或 Pod 无法直连外网（防火墙/无路由）时，让流量经一台「能上网」的机器转发。
- **何时不需要**：若能放行节点/Pod 的出站 443（方式 A），就不必搞代理。

**常见获取方式：**

1. **公司/学校已有代理**  
   若你在内网，问运维是否提供 HTTP 代理（地址如 `proxy.company.com:3128`）。有的话把该地址填到 `HTTPS_PROXY` 即可，且需保证 k3s 节点能访问该地址。

2. **自己搭一台（推荐：有一台能上网的 Linux 时）**  
   在一台**能访问互联网**的机器上（可以是你的笔记本、家里另一台电脑、或一台 VPS）装一个轻量 HTTP 代理，让 k3s 节点能访问这台机器的 IP 和端口。

   **用 TinyProxy（简单）：**

   ```bash
   # 以 Rocky/RHEL/CentOS 为例
   sudo dnf install -y tinyproxy
   # 允许来自 k3s 节点 IP 的请求（或 0.0.0.0 仅内网测试）
   echo 'Allow 10.42.0.0/16' | sudo tee -a /etc/tinyproxy/tinyproxy.conf
   echo 'Allow 192.168.0.0/16' | sudo tee -a /etc/tinyproxy/tinyproxy.conf
   sudo systemctl enable --now tinyproxy
   ```

   默认监听 8888。若这台机器 IP 是 `192.168.1.100`，则在 Backend 的 ConfigMap/Secret 里设：

   ```yaml
   HTTPS_PROXY: "http://192.168.1.100:8888"
   HTTP_PROXY: "http://192.168.1.100:8888"
   ```

   确保 k3s 节点能访问 `192.168.1.100:8888`（防火墙放行 8888）。

   **用 Squid（功能更多）：**

   ```bash
   sudo dnf install -y squid
   # 编辑 /etc/squid/squid.conf，允许你的内网网段
   # acl local_net src 10.42.0.0/16
   # acl local_net src 192.168.0.0/16
   # http_access allow local_net
   sudo systemctl enable --now squid
   ```

   默认端口 3128，对应 `HTTPS_PROXY: "http://<该机IP>:3128"`。

3. **没有可用的代理时**  
   若既没有现成代理，也没有能上网的机器可装代理，就只能从网络侧解决：在 k3s 节点或上游路由器/防火墙上放行出站 443（方式 A），或检查默认网关/路由是否导致「no route to host」。

**小结**：`your-proxy-host` = 运行代理服务的那台机器的 **IP 或域名**，`3128`/`8888` 等 = **代理监听端口**。Backend 通过 `HTTPS_PROXY` 把 Stripe 的请求发到该地址，由代理代为访问互联网。

**在节点上跑代理（Pod 出网 NAT 仍不通时）**

当「节点能连 Stripe、Pod 不能」且 MASQUERADE / firewalld masquerade 仍无效时，可在 **k3s 节点本机** 跑一个 HTTP 代理：节点能直连 Stripe，Pod 只需连到节点 IP，不经过出网 NAT，即可用。

1. **在 k3s 节点上**（如 msd6200）安装并配置 TinyProxy：

   ```bash
   sudo dnf install -y tinyproxy
   # 只允许 Pod 网段（避免对外暴露）
   echo 'Allow 10.42.0.0/16' | sudo tee -a /etc/tinyproxy/tinyproxy.conf
   echo 'Allow 127.0.0.1' | sudo tee -a /etc/tinyproxy/tinyproxy.conf
   # 必须监听 0.0.0.0，否则 Pod 连节点 IP 会 connection refused（默认或部分发行版为 Listen 127.0.0.1）
   sudo sed -i '/^Listen /d' /etc/tinyproxy/tinyproxy.conf
   echo 'Listen 0.0.0.0' | sudo tee -a /etc/tinyproxy/tinyproxy.conf
   sudo systemctl enable --now tinyproxy
   # 确认：应看到 0.0.0.0:8888，若为 127.0.0.1:8888 则 Pod 连不到
   ss -tlnp | grep 8888
   ```

2. **放行节点本机 8888 入站**（firewalld）：

   ```bash
   sudo firewall-cmd --permanent --add-port=8888/tcp
   sudo firewall-cmd --reload
   ```

3. **Backend 使用节点 IP 作为代理**：把 `HTTPS_PROXY` / `HTTP_PROXY` 设为节点 IP（如 `208.122.213.192`）：

   - ConfigMap：`backend/k8s/configmap.yaml` 里设 `HTTPS_PROXY: "http://208.122.213.192:8888"`、`HTTP_PROXY: "http://208.122.213.192:8888"`（IP 换成你的节点 IP）。
   - 或 Secret / `.env` 里设同名变量。

4. **应用并重启**：

   ```bash
   kubectl apply -f backend/k8s/configmap.yaml
   kubectl rollout restart deployment/twixter-backend
   ```

5. **验证**：在 Pod 内应能连节点 8888：`kubectl run -it --rm debug --image=busybox --restart=Never -- sh -c "nc -zv 208.122.213.192 8888 || true"`（IP 换成你的节点 IP）。成功后 Stripe 请求会经节点上的 TinyProxy 转发。

   **若仍报 connection refused**：Backend 与代理同节点时，用节点公网 IP 可能走「本机连本机」被拒。可改用 **Pod 的默认网关**（多为节点在集群内的 IP，k3s 常见为 `10.42.0.1`）：在 ConfigMap/Secret 里设 `HTTPS_PROXY=http://10.42.0.1:8888`（单节点时一般为 10.42.0.1）。多节点时需让 Backend 连到**其所在节点**的网关 IP，或继续用节点公网 IP 并确认该节点防火墙上 8888 已对 10.42.0.0/16 放行。

---

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
