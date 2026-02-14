# 测试 OpenRouter API 是否可用

当聊天流式接口返回 200 但界面没有助手回复时，可先确认 OpenRouter 与密钥是否正常。

## 1. 确认环境变量

在 `backend/.env` 中设置（不要有引号、前后空格）：

```env
OPENROUTER_API_KEY=sk-or-v1-你的密钥
OPENROUTER_MODEL=cognitivecomputations/dolphin-mistral-24b-venice-edition:free
```

- API Key 在 [OpenRouter Keys](https://openrouter.ai/keys) 创建。
- 模型页：<https://openrouter.ai/cognitivecomputations/dolphin-mistral-24b-venice-edition:free>  
  若模型下线或限流，可换其他模型（如 `openai/gpt-3.5-turbo`）再试。

## 2. 用 curl 直接测流式接口

在终端执行（把 `YOUR_API_KEY` 换成真实密钥）：

```bash
curl -s -N -X POST "https://openrouter.ai/api/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "cognitivecomputations/dolphin-mistral-24b-venice-edition:free",
    "messages": [{"role": "user", "content": "Say hello in one word."}],
    "stream": true
  }'
```

- **有正常回复**：会看到多行 `data: {"choices":[...]}` 和最后的 `data: [DONE]`，说明 API 和密钥正常，问题多半在应用侧（解析或前端）。
- **401**：密钥错误或未设置。
- **404 / 502**：模型或 OpenRouter 暂时不可用，可换模型或稍后重试。
- **无输出或长时间无 data**：网络或 OpenRouter 侧问题。

## 3. 看后端日志

已对 OpenRouter 客户端加了诊断：当流式返回**零内容**时，后端会打日志，例如：

- `[openrouter] stream returned no content; model=...`  
  表示 OpenRouter 返回了 200 但没有任何有效 content 块（或格式与预期不符）。
- `[openrouter] stream returned no content; first raw line (sample): ...`  
  会带上收到的第一行原始内容，便于核对是否换了格式。

重启后端后再发一条聊天消息，看控制台是否出现上述日志，可帮助判断是 API 无内容还是解析问题。
