package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// StreamCompletion 将 prompt 作为单条 user 消息调用 NinjaChat /chat（非流式），通过 onChunk 一次性推送全文；返回完整 assistant 文本。
func (c *NinjaChatClient) StreamCompletion(ctx context.Context, prompt string, onChunk func(text string)) (full string, err error) {
	if c == nil {
		return "", fmt.Errorf("ninjachat client not configured")
	}
	body := ninjaChatRequest{
		Messages:    []ninjaChatMessage{{Role: "user", Content: prompt}},
		Model:       c.model,
		Temperature: 0.85,
		MaxTokens:   150,
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
	if err := json.Unmarshal(b, &parsed); err != nil {
		return "", fmt.Errorf("ninjachat parse response: %w", err)
	}
	full = strings.TrimSpace(parsed.Message.Content)
	if full == "" {
		return "", nil
	}
	full = stripRoleLabels(full)
	if onChunk != nil {
		onChunk(full)
	}
	return full, nil
}
