# k3s / Kubernetes 上为 Ingress 启用 HTTPS（解决 “Your connection isn’t secure”）

浏览器提示 **“Your connection to this site isn't secure”** 表示当前没有受信任的 TLS 证书（或证书未正确挂到 Ingress）。按下面步骤在 k3s 上用 **cert-manager + Let’s Encrypt** 自动签发并挂载证书。

## 前置条件

- 集群已安装 **nginx ingress**（`ingressClassName: nginx`）。
- 域名 **www.twixter.store**、**api.twixter.store** 的 DNS 已指向集群入口（LoadBalancer IP 或 NodePort 对应 IP），且 **80 / 443** 可从公网访问（Let’s Encrypt HTTP-01 需要访问你域名的 80 端口）。

## 1. 安装 cert-manager（若未安装）

k3s 默认不带 cert-manager，需要单独安装。**无需 Helm**，用 `kubectl` 即可：

```bash
# 官方 manifest（含 CRD），见 https://cert-manager.io/docs/installation/kubectl/
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.4/cert-manager.yaml
```

确认 Pod 就绪（通常会有 cert-manager、cainjector、webhook 三个）：

```bash
kubectl get pods -n cert-manager
```

若已安装 Helm，也可用 Helm 安装（可选）：

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
kubectl create namespace cert-manager
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --set installCRDs=true
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

### 6.1 错误：`lookup acme-v02.api.letsencrypt.org on 10.43.0.10:53: server misbehaving`

这是**集群内 DNS 解析失败**：cert-manager 用集群 DNS（k3s 的 CoreDNS）解析 Let’s Encrypt，上游 DNS 异常或不可达会导致 “server misbehaving”，证书无法签发。

**处理步骤：**

1. **在节点上测 DNS**（在运行 k3s 的机器上执行）：
   ```bash
   nslookup acme-v02.api.letsencrypt.org
   # 或
   dig acme-v02.api.letsencrypt.org
   ```
   若节点上也解析失败，先修节点 DNS（如 `/etc/resolv.conf` 改为 `nameserver 8.8.8.8` 或 `1.1.1.1`）。

2. **让 CoreDNS 使用公网 DNS**（节点能解析但 Pod 内不能时）：  
   k3s 的 CoreDNS 配置在 `kube-system` 的 ConfigMap `coredns` 里。编辑其 `Corefile`，在 `.:53` 的 `forward` 里使用公网 DNS，例如：
   ```bash
   kubectl edit configmap coredns -n kube-system
   ```
   在 `Corefile` 里把 `forward . /etc/resolv.conf` 改为：
   ```
   forward . 8.8.8.8 1.1.1.1
   ```
   保存后重启 CoreDNS Pod 使配置生效：
   ```bash
   kubectl rollout restart deployment coredns -n kube-system
   # 若 coredns 是 DaemonSet：
   kubectl rollout restart daemonset coredns -n kube-system
   ```

3. **确认 Pod 内能解析**：
   ```bash
   kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup acme-v02.api.letsencrypt.org
   ```
   能解析到 IP 后，cert-manager 会重试并有望签发证书。可再查看：
   ```bash
   kubectl get certificate -n default
   kubectl logs -n cert-manager -l app=cert-manager --tail=30
   ```

### 6.2 错误：`read udp ...->8.8.8.8:53: read: no route to host` 或 `i/o timeout`

表示 **Pod 无法访问公网 DNS**（8.8.8.8 / 1.1.1.1）：防火墙或路由阻止了集群到外网的 UDP/TCP 53。

**处理步骤：**

1. **先把 CoreDNS 改回用节点 DNS**，避免集群内解析全挂：
   ```bash
   kubectl get configmap coredns -n kube-system -o yaml > coredns.yaml
   # 编辑 coredns.yaml，把 forward . 8.8.8.8 1.1.1.1 改回 forward . /etc/resolv.conf
   sed -i 's|forward \. 8.8.8.8 1.1.1.1|forward . /etc/resolv.conf|g' coredns.yaml
   sed -i '/^\s*resourceVersion:/d' coredns.yaml
   sed -i '/^\s*uid:/d' coredns.yaml
   kubectl apply -f coredns.yaml
   kubectl rollout restart deployment coredns -n kube-system
   ```

2. **在 k3s 节点上**测 DNS 和路由：
   ```bash
   nslookup acme-v02.api.letsencrypt.org
   cat /etc/resolv.conf
   ping -c 1 8.8.8.8
   ```
   - 若**节点能解析**：说明节点有可用 DNS，但 Pod 出网被拦。需要放行 **从 Pod 网段（如 10.42.0.0/16）到外网的 UDP/TCP 53**，或放行节点做 NAT 后的出站 53（依你环境防火墙/安全组配置）。
   - 若**节点也不能解析 / 不能 ping 8.8.8.8**：先修节点网络（路由、防火墙、`/etc/resolv.conf`），再考虑证书。

3. **可选**：若你有一个 **Pod 能访问的 DNS**（例如节点所在局域网的网关 192.168.x.1 或内网 DNS），可把 CoreDNS 的 `forward` 改成该 IP，例如 `forward . 192.168.1.1`，再重启 CoreDNS。cert-manager 能解析 Let's Encrypt 后，证书才会 Ready。若**节点上** `nslookup` 和 `cat /etc/resolv.conf` 正常，可把 CoreDNS 的 forward 改成节点用的 nameserver（如 `forward . 208.74.148.200 208.74.148.216`），Pod 若能访问该 DNS，证书即可签发。

4. **Pod 仍无法访问任何外网 DNS 时**：可让 CoreDNS 使用节点网络出网，这样转发请求从节点发出，节点能解析则证书可签发。在运行 k3s 的节点上执行：
   ```bash
   kubectl patch deployment coredns -n kube-system -p '{"spec":{"template":{"spec":{"hostNetwork":true}}}}'
   kubectl rollout restart deployment coredns -n kube-system
   ```
   若 rollout 一直卡在 “0 of 1 updated replicas are available” 且 `kubectl describe pod` 显示 **Readiness probe failed: HTTP probe failed with statuscode: 404**，说明就绪探针访问的 :8181/ready 返回 404，可把就绪探针改为与存活探针相同的 health 端口后再试：
   ```bash
   kubectl patch deployment coredns -n kube-system --type='json' -p='[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe/httpGet","value":{"path":"/health","port":8080}}]'
   kubectl rollout status deployment coredns -n kube-system
   ```
   注意：k3s 重启后可能恢复 CoreDNS 配置，若 forward 或 hostNetwork 被还原，需重新修改或 patch。

### 6.3 其他情况

- **Challenge 失败**：确认 80 端口可从公网访问你的域名，且 nginx ingress 的 80 入口指向当前集群（无其他反向代理或防火墙拦截）。
- **限速**：Let’s Encrypt 生产环境有速率限制；调试时可先用 [staging 环境](https://cert-manager.io/docs/configuration/acme/#staging)（证书不被浏览器信任，仅用于验证流程）。
- **查看 cert-manager 日志**：

```bash
kubectl logs -n cert-manager -l app=cert-manager
kubectl logs -n cert-manager -l app=webhook
```

## 7. 方案 A：放行 Pod 出站（推荐）

若 cert-manager 日志出现 `dial tcp ...:443: connect: no route to host`，说明 **Pod 访问外网 443/80 被拦**。在 **运行 k3s 的节点**上放行 Pod 网段出站即可。

**先确认 Pod 网段**（k3s 默认多为 `10.42.0.0/16`）：

```bash
# 查看节点 Pod CIDR
kubectl get nodes -o wide
kubectl get node -o jsonpath='{.items[0].spec.podCIDR}'
# 若无输出，看 k3s 配置：sudo cat /etc/rancher/k3s/config.yaml 或 --cluster-cidr
```

下面用 `POD_CIDR=10.42.0.0/16`；若你集群不同，替换成实际值。

### 7.1 用 iptables 放行（节点上执行，需 root）

```bash
# 放行 Pod 网段出站：80（HTTP-01）、443（HTTPS/ACME）、53（DNS）
POD_CIDR="10.42.0.0/16"
sudo iptables -I FORWARD 1 -s "$POD_CIDR" -p tcp --dport 80  -j ACCEPT
sudo iptables -I FORWARD 1 -s "$POD_CIDR" -p tcp --dport 443 -j ACCEPT
sudo iptables -I FORWARD 1 -s "$POD_CIDR" -p udp --dport 53  -j ACCEPT
# 允许回包（ESTABLISHED,RELATED 一般已有，若有 DROP 策略可显式加）
sudo iptables -I FORWARD 1 -d "$POD_CIDR" -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
```

**持久化**（重启后仍生效，按你系统选一种）：

```bash
# Debian/Ubuntu：安装 iptables-persistent 后保存
sudo apt-get install -y iptables-persistent
sudo netfilter-persistent save

# 或手动保存到文件，开机脚本恢复
sudo iptables-save | sudo tee /etc/iptables.rules
# 在 /etc/rc.local 或 systemd 服务里加：iptables-restore < /etc/iptables.rules
```

### 7.2 用 firewalld 放行（节点上执行，需 root）

```bash
POD_CIDR="10.42.0.0/16"
sudo firewall-cmd --permanent --direct --add-rule ipv4 filter FORWARD 0 -s "$POD_CIDR" -p tcp --dport 80  -j ACCEPT
sudo firewall-cmd --permanent --direct --add-rule ipv4 filter FORWARD 0 -s "$POD_CIDR" -p tcp --dport 443 -j ACCEPT
sudo firewall-cmd --permanent --direct --add-rule ipv4 filter FORWARD 0 -s "$POD_CIDR" -p udp --dport 53  -j ACCEPT
sudo firewall-cmd --permanent --direct --add-rule ipv4 filter FORWARD 0 -d "$POD_CIDR" -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
sudo firewall-cmd --reload
```

### 7.3 云安全组 / 外部防火墙

若节点在云上或前面还有防火墙，需在**安全组或防火墙上**放行：

- **出站**：源 = 节点 IP（或 Pod 网段若不经 NAT），目标 = 0.0.0.0/0，端口 **TCP 80、443**，**UDP 53**。

放行后等 1～2 分钟，再查证书和日志：

```bash
kubectl get certificate -n default
kubectl logs -n cert-manager -l app=cert-manager --tail=20
```

---

## 8. 未使用 cert-manager 时（自签名等）

若暂时不用 Let’s Encrypt，可手动创建 TLS Secret，并在 Ingress 的 `tls.secretName` 中引用该 Secret；浏览器会提示不受信任，需手动添加例外。生产环境建议使用 cert-manager + Let’s Encrypt。
