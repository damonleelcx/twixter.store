package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	openRouterBaseURL = "https://openrouter.ai/api/v1"
	defaultOpenRouterModel = "cognitivecomputations/dolphin-mistral-24b-venice-edition:free"
)

// OpenRouterClient 调用 OpenRouter Chat Completions API（流式）。
// 使用环境变量 OPENROUTER_API_KEY 鉴权，OPENROUTER_MODEL 指定模型（默认 dolphin-mistral-24b-venice-edition:free）。
type OpenRouterClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenRouterClient 创建 OpenRouter 客户端。apiKey 为空则返回 nil。
func NewOpenRouterClient() *OpenRouterClient {
	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if apiKey == "" {
		return nil
	}
	model := strings.TrimSpace(os.Getenv("OPENROUTER_MODEL"))
	if model == "" {
		model = defaultOpenRouterModel
	}
	return &OpenRouterClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 2 * time.Minute,
			},
		},
	}
}

// openRouterChatRequest OpenRouter /chat/completions 请求体（与 OpenAI 兼容）
type openRouterChatRequest struct {
	Model       string                  `json:"model"`
	Messages    []openRouterChatMessage `json:"messages"`
	Stream      bool                    `json:"stream"`
	Temperature float64                 `json:"temperature,omitempty"`
	MaxTokens   int                     `json:"max_tokens,omitempty"`
}

type openRouterChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openRouterStreamChunk 流式响应中的单条 data
type openRouterStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// StreamCompletion 将 prompt 作为单条 user 消息调用 OpenRouter 流式 chat/completions，通过 onChunk 回调推送；返回完整 assistant 文本。
func (c *OpenRouterClient) StreamCompletion(ctx context.Context, prompt string, onChunk func(text string)) (full string, err error) {
	if c == nil {
		return "", fmt.Errorf("openrouter client not configured")
	}
	body := openRouterChatRequest{
		Model:    c.model,
		Messages: []openRouterChatMessage{{Role: "user", Content: prompt}},
		Stream:   true,
		Temperature: 0.7,
		MaxTokens:   2048,
	}
	jb, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openRouterBaseURL+"/chat/completions", bytes.NewReader(jb))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openrouter request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("openrouter returned %d: %s", resp.StatusCode, string(b))
	}

	var fullBuilder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	const dataPrefix = "data: "

	var firstRawLine string
	var streamBuf strings.Builder
	prefixDone := false
	rolePrefixes := []string{"assistant: ", "assistant:", "Assistant: ", "Assistant:", "user: ", "user:", "User: ", "User:"}
	var ohNameBuf strings.Builder
	ohNameDone := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, ": ") {
			// OpenRouter keepalive comment, e.g. ": OPENROUTER PROCESSING"
			continue
		}
		if !strings.HasPrefix(line, dataPrefix) {
			if firstRawLine == "" {
				firstRawLine = line
			}
			continue
		}
		payload := strings.TrimSpace(line[len(dataPrefix):])
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var chunk openRouterStreamChunk
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			if firstRawLine == "" {
				firstRawLine = payload
			}
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}
		fullBuilder.WriteString(content)
		if !prefixDone {
			streamBuf.WriteString(content)
			raw := streamBuf.String()
			lower := strings.ToLower(raw)
			stripLen := 0
			for _, p := range rolePrefixes {
				pl := strings.ToLower(p)
				if strings.HasPrefix(lower, pl) && len(pl) > stripLen {
					stripLen = len(pl)
				}
			}
			if stripLen > 0 {
				prefixDone = true
				if rest := strings.TrimSpace(raw[stripLen:]); rest != "" && onChunk != nil {
					onChunk(rest)
				}
				streamBuf.Reset()
				continue
			}
			if len(raw) >= 16 || strings.Contains(lower, "\n") {
				prefixDone = true
				ohNameBuf.WriteString(raw)
				streamBuf.Reset()
			}
			continue
		}
		if !ohNameDone {
			ohNameBuf.WriteString(content)
			raw := ohNameBuf.String()
			if len(raw) >= 8 || strings.Contains(raw, "\n") {
				if idx := reOhNamePrefix.FindStringIndex(raw); idx != nil && idx[0] == 0 {
					rest := raw[idx[1]:]
					ohNameBuf.Reset()
					ohNameDone = true
					if rest != "" && onChunk != nil {
						onChunk(rest)
					}
				} else if len(raw) >= 40 || strings.Contains(raw, "\n") {
					ohNameDone = true
					if onChunk != nil {
						onChunk(raw)
					}
					ohNameBuf.Reset()
				}
			}
			continue
		}
		if onChunk != nil {
			onChunk(content)
		}
	}
	if err := scanner.Err(); err != nil {
		return stripRoleLabels(fullBuilder.String()), fmt.Errorf("openrouter stream read: %w", err)
	}
	if ohNameBuf.Len() > 0 && !ohNameDone {
		raw := ohNameBuf.String()
		if idx := reOhNamePrefix.FindStringIndex(raw); idx != nil && idx[0] == 0 {
			raw = raw[idx[1]:]
		}
		if raw != "" && onChunk != nil {
			onChunk(raw)
		}
	}
	if fullBuilder.Len() == 0 {
		if firstRawLine != "" {
			log.Printf("[openrouter] stream returned no content; first raw line (sample): %s", truncate(firstRawLine, 200))
		} else {
			log.Printf("[openrouter] stream returned no content; model=%s", c.model)
		}
	}
	return stripRoleLabels(fullBuilder.String()), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
