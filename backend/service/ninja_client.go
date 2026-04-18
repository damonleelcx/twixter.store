package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	ninjaChatBaseURL = "https://ninjachat.ai/api/v1"
	defaultNinjaModel = "ninja-1"
)

// NinjaChatClient 调用 NinjaChat /chat API（非流式，兼容 LLMStreamer）。
// 使用环境变量 NINJACHAT_API_KEY 鉴权，NINJACHAT_MODEL 指定模型（默认 ninja-1）。
type NinjaChatClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewNinjaChatClient 创建 NinjaChat 客户端。apiKey 为空则返回 nil。
func NewNinjaChatClient() *NinjaChatClient {
	apiKey := strings.TrimSpace(os.Getenv("NINJACHAT_API_KEY"))
	if apiKey == "" {
		return nil
	}
	model := strings.TrimSpace(os.Getenv("NINJACHAT_MODEL"))
	if model == "" {
		model = defaultNinjaModel
	}
	return &NinjaChatClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 2 * time.Minute,
			},
		},
	}
}

type ninjaChatRequest struct {
	Messages    []ninjaChatMessage `json:"messages"`
	Model       string             `json:"model"`
	Temperature float64            `json:"temperature,omitempty"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
}

type ninjaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ninjaChatResponse NinjaChat 实际返回格式
type ninjaChatResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Usage    interface{} `json:"usage,omitempty"`
	Cost     interface{} `json:"cost,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// ninjaAssistantTextFromJSON 从 Ninja /chat 或兼容形态中取出助手正文（message.content 可能为 string 或块数组；部分网关返回 choices[].message）。
func ninjaAssistantTextFromJSON(b []byte) (string, bool) {
	var root map[string]interface{}
	if err := json.Unmarshal(b, &root); err != nil {
		return "", false
	}
	if s, ok := stringFromMessageContent(root["message"]); ok {
		return s, true
	}
	if choices, ok := root["choices"].([]interface{}); ok && len(choices) > 0 {
		if ch0, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := ch0["message"].(map[string]interface{}); ok {
				if s, ok := stringFromMessageContent(msg); ok {
					return s, true
				}
			}
			if s, ok := ch0["text"].(string); ok && strings.TrimSpace(s) != "" {
				return s, true
			}
		}
	}
	if s, ok := root["text"].(string); ok && strings.TrimSpace(s) != "" {
		return s, true
	}
	if s, ok := root["response"].(string); ok && strings.TrimSpace(s) != "" {
		return s, true
	}
	return "", false
}

// stringFromMessageContent 解析 message 对象里的 content（string 或多段 {type,text}）。
func stringFromMessageContent(msg interface{}) (string, bool) {
	m, ok := msg.(map[string]interface{})
	if !ok {
		return "", false
	}
	raw, ok := m["content"]
	if !ok || raw == nil {
		return "", false
	}
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return "", false
		}
		return s, true
	case []interface{}:
		var parts []string
		for _, item := range v {
			part, ok := contentPartToString(item)
			if ok && part != "" {
				parts = append(parts, part)
			}
		}
		out := strings.TrimSpace(strings.Join(parts, ""))
		if out == "" {
			return "", false
		}
		return out, true
	case map[string]interface{}:
		// NinjaChat 等网关有时把 assistant content 直接解析为 JSON 对象（如 {"title":"...","description":"..."}），而非字符串。
		jb, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		s := strings.TrimSpace(string(jb))
		if s == "" || s == "null" || s == "{}" {
			return "", false
		}
		return s, true
	default:
		return "", false
	}
}

func contentPartToString(item interface{}) (string, bool) {
	switch v := item.(type) {
	case string:
		return v, true
	case map[string]interface{}:
		if t, ok := v["text"].(string); ok {
			return t, true
		}
		if t, ok := v["content"].(string); ok {
			return t, true
		}
	}
	return "", false
}

func ninjaResponseTopKeys(root map[string]interface{}, max int) string {
	if len(root) == 0 {
		return ""
	}
	keys := make([]string, 0, len(root))
	for k := range root {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > max {
		keys = keys[:max]
	}
	return strings.Join(keys, ",")
}

// complete 调用 NinjaChat /chat（非流式），返回助手全文。
func (c *NinjaChatClient) complete(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
	if c == nil {
		return "", fmt.Errorf("ninjachat client not configured")
	}
	if maxTokens <= 0 {
		maxTokens = 150
	}
	body := ninjaChatRequest{
		Messages:    []ninjaChatMessage{{Role: "user", Content: prompt}},
		Model:       c.model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	jb, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ninjaChatBaseURL+"/chat", bytes.NewReader(jb))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ninjachat request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("ninjachat returned %d: %s", resp.StatusCode, string(b))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ninjachat read body: %w", err)
	}
	var parsed ninjaChatResponse
	_ = json.Unmarshal(b, &parsed)

	full := strings.TrimSpace(parsed.Message.Content)
	if full == "" {
		if alt, ok := ninjaAssistantTextFromJSON(b); ok {
			full = strings.TrimSpace(alt)
		}
	}
	if full == "" {
		var root map[string]interface{}
		_ = json.Unmarshal(b, &root)
		hint := ""
		if root != nil {
			hint = fmt.Sprintf(" (top-level keys: %s)", ninjaResponseTopKeys(root, 12))
		}
		snippet := string(b)
		if len(snippet) > 280 {
			snippet = snippet[:280] + "…"
		}
		return "", fmt.Errorf("ninjachat empty assistant message%s; body≈ %q", hint, snippet)
	}
	return stripRoleLabels(full), nil
}

// Chat 单次补全，可指定 max_tokens（需比默认 Stream 更长输出时使用）。
func (c *NinjaChatClient) Chat(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return c.complete(ctx, prompt, 0.75, maxTokens)
}

// StreamCompletion 将 prompt 作为单条 user 消息调用 NinjaChat /chat（非流式），通过 onChunk 一次性推送全文；返回完整 assistant 文本。
func (c *NinjaChatClient) StreamCompletion(ctx context.Context, prompt string, onChunk func(text string)) (full string, err error) {
	full, err = c.complete(ctx, prompt, 0.85, 150)
	if err != nil {
		return "", err
	}
	if onChunk != nil && full != "" {
		onChunk(full)
	}
	return full, nil
}
