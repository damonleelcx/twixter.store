package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMClient 调用自托管 LLM（如 llama.cpp server）的流式接口
type LLMClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewLLMClient 创建 LLM 客户端；baseURL 如 http://llm:8081
func NewLLMClient(baseURL string) *LLMClient {
	if baseURL == "" {
		return nil
	}
	return &LLMClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   0, // 流式响应不设总超时
			Transport: &http.Transport{ResponseHeaderTimeout: 30 * time.Second},
		},
	}
}

// LlamaCppCompletionRequest llama.cpp server 常见请求体
type LlamaCppCompletionRequest struct {
	Prompt      string  `json:"prompt"`
	Stream      bool   `json:"stream"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int    `json:"n_predict,omitempty"`
}

// StreamCompletion 流式调用 /completion，每行 JSON 通过 onChunk 回调；返回完整 assistant 文本
func (c *LLMClient) StreamCompletion(ctx context.Context, prompt string, onChunk func(text string)) (full string, err error) {
	if c == nil {
		return "", fmt.Errorf("llm client not configured")
	}
	body := LlamaCppCompletionRequest{Prompt: prompt, Stream: true, Temperature: 0.7, MaxTokens: 2048}
	jb, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/completion", bytes.NewReader(jb))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, string(b))
	}

	var fullBuilder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk struct {
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(line), &chunk) != nil {
			continue
		}
		if chunk.Content != "" {
			fullBuilder.WriteString(chunk.Content)
			if onChunk != nil {
				onChunk(chunk.Content)
			}
		}
	}
	return fullBuilder.String(), scanner.Err()
}
