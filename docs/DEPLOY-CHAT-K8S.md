# 聊天栈生产部署指南（Docker + Kubernetes）

适用于：自托管 NSFW-capable LLM、Go 后端（SSE 流式）、MongoDB 消息历史、PostgreSQL 人格存储、Next.js 前端、单节点/小集群 k3s。

## 1. 目标架构

```
                    Ingress (NGINX / Traefik)
                              │
              ┌───────────────┴───────────────┐
              │                               │
        Next.js UI                      Go Backend (SSE + Memory)
              │                               │
              │                    ┌──────────┴──────────┐
              │                    │                     │
              │              PostgreSQL              MongoDB
              │              (人格/名字)              (消息历史)
              │                    │                     │
              │                    └──────────┬──────────┘
              │                               │
              │                        LLM Inference Pod
              │                        (llama.cpp / TGI)
              └───────────────────────────────┘
```

- **首次进入聊天**：Agent 要求用户填写名字与人格（`tone`, `speaking_style`, `boundaries`, `quirks`, `emotional_range`），写入 **PostgreSQL**。
- **消息历史**：短期会话与消息列表写入 **MongoDB**，并配置 TTL 自动过期。

## 2. 聊天 API 行为

| 接口 | 说明 |
|-----|------|
| `GET /api/chat/session?session_id=xxx` | 获取会话状态；若未填写人格则返回 `need_onboarding: true`，前端应引导用户提交名字与人格。 |
| `POST /api/chat/onboarding` | 提交 `name` + `personality`（JSON），写入 Postgres。 |
| `POST /api/chat/stream` | 发送消息并接收 SSE 流式回复；若未先完成 onboarding 则返回 `428 NEED_PERSONA`。 |

人格 JSON 示例：

```json
{
  "tone": "confident, playful",
  "speaking_style": "short teasing sentences",
  "boundaries": "never breaks character",
  "quirks": ["sarcastic humor", "slow reveals"],
  "emotional_range": "intimate but controlled"
}
```

## 3. Kubernetes（k3s）部署

### 3.1 安装 k3s

```bash
curl -sfL https://get.k3s.io | sh -
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
kubectl get nodes
```

### 3.2 已包含的 k8s 资源（backend/k8s）

- **MongoDB**：`mongodb-statefulset.yaml` + `mongodb-service.yaml`（消息历史，10Gi PVC）
- **LLM**：`llm-deployment.yaml` + `llm-service.yaml`（示例为 llama.cpp server，可按需替换镜像与资源）
- **Backend**：ConfigMap 中已配置 `MONGO_URI=mongodb://mongodb:27017`、`LLM_ENDPOINT=http://llm:8081`
- **Ingress**：已为 SSE 设置 `proxy-buffering: off`、长超时，保证 `/api/chat/stream` 流式正常

一键应用（在仓库根目录，且 `backend/k8s/.env` 已配置）：

```bash
kubectl apply -k backend/k8s/
```

### 3.3 SSE 与副本数

- 聊天流式接口建议 **Backend replicas=1** 或为 Ingress 配置 **session affinity**，避免同一会话被负载到不同 Pod。
- 已通过 Ingress annotation 关闭 proxy buffering，Traefik（k3s 默认）本身对 SSE 友好。

## 4. MongoDB 消息设计（简要）

- **短期会话**：`chat.sessions`，字段含 `session_id`、`user_id`、`messages[]`、`expires_at`。
- TTL 索引：`expires_at` + `expireAfterSeconds: 0`，由后端在 `store/mongo_chat.go` 中创建。

## 5. 自托管 LLM 镜像示例（llama.cpp）

见 `backend/k8s/llm-deployment.yaml`；也可使用项目内提供的 Dockerfile 自建镜像并挂载模型 PVC：

```dockerfile
# 见 backend/deploy/LLM_DOCKERFILE.example
FROM ghcr.io/ggerganov/llama.cpp:latest
COPY ./models /models
EXPOSE 8081
CMD ["./server", "-m", "/models/model.gguf", "--port", "8081"]
```

## 6. 生产检查清单

- [ ] 聊天接口使用 Sticky Session 或 replicas=1
- [ ] Ingress 对 `/api/chat/stream` 关闭 proxy buffering
- [ ] MongoDB 使用 PVC 持久化
- [ ] 人格与 prompt 仅存服务端（Postgres + 后端逻辑），前端不直连模型
- [ ] LLM 与 Backend 分离部署，资源按需限制（如 LLM 8 CPU / 32Gi）
