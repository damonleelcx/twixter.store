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
   ```
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
