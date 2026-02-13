package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
			Timeout: 0, // 流式响应不设总超时
			Transport: &http.Transport{
				// 7B+ 模型在 CPU 上首 token 可能需 1–3 分钟，等待响应头的超时需足够长
				ResponseHeaderTimeout: 5 * time.Minute,
			},
		},
	}
}

// LlamaCppCompletionRequest llama.cpp server 常见请求体
type LlamaCppCompletionRequest struct {
	Prompt      string   `json:"prompt"`
	Stream      bool     `json:"stream"`
	Temperature float64  `json:"temperature,omitempty"`
	MaxTokens   int      `json:"n_predict,omitempty"`
	Stop        []string `json:"stop,omitempty"` // 遇到即停止，避免生成下一轮的 user:/assistant: 等
}

// 移除 markdown 链接 [text](url) 和裸 URL，避免流式输出中出现推广链接
var (
	reMarkdownLink = regexp.MustCompile(`\[[^\]]*\]\(https?://[^\)]*\)`)
	reBareURL      = regexp.MustCompile(`https?://[^\s\)\]\>]+`)
)

func removeLinks(s string) string {
	s = reMarkdownLink.ReplaceAllString(s, "")
	s = reBareURL.ReplaceAllString(s, "")
	return s
}

// 截断推广句：从 "(If you want to chat with" 起的内容一律不输出
var rePromoPhrase = regexp.MustCompile(`(?i)\(If you want to chat with`)

func stripPromoPhrase(s string) string {
	if idx := rePromoPhrase.FindStringIndex(s); idx != nil {
		return strings.TrimRight(s[:idx[0]], " (\n")
	}
	return s
}

// StreamCompletion 流式调用 /completion，每行 JSON 通过 onChunk 回调；返回完整 assistant 文本
func (c *LLMClient) StreamCompletion(ctx context.Context, prompt string, onChunk func(text string)) (full string, err error) {
	if c == nil {
		return "", fmt.Errorf("llm client not configured")
	}
	body := LlamaCppCompletionRequest{
		Prompt:      prompt,
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   2048,
		Stop:        []string{"\nuser:", "\nassistant:", "user:", "assistant:", "(If you want to chat with", "you can do so [here]", "you can do so here"},
	}
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
	const dataPrefix = "data: "
	// 过滤流：去掉开头的 "assistant:"，遇到 "\nuser:" / "\nassistant:" 停止转发；去掉链接；按批发送减少 SSE 事件
	var streamBuf strings.Builder
	var pendingLink strings.Builder   // 用于链接过滤的累积
	var lastEmittedCleanLen int       // 已发出的 cleaned 长度
	var emitBuf string                // 批缓冲，凑够再 onChunk
	const emitBatchSize = 24          // 至少 N 字符或含换行再推一次，减少 SSE 事件
	prefixDone := false
	stopForward := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, dataPrefix) {
			line = strings.TrimSpace(line[len(dataPrefix):])
		}
		if line == "" || line == "[DONE]" {
			continue
		}
		var chunk struct {
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(line), &chunk) != nil {
			continue
		}
		content := chunk.Content
		if content == "" {
			continue
		}
		if stopForward {
			continue
		}
		streamBuf.WriteString(content)
		raw := streamBuf.String()
		if !prefixDone {
			lower := strings.ToLower(raw)
			for _, p := range []string{"assistant: ", "assistant:"} {
				if strings.HasPrefix(lower, p) {
					streamBuf.Reset()
					rest := raw[len(p):]
					prefixDone = true
					if rest != "" {
						pendingLink.WriteString(rest)
						cleaned := stripPromoPhrase(removeLinks(pendingLink.String()))
						if len(cleaned) < lastEmittedCleanLen {
							lastEmittedCleanLen = len(cleaned)
						}
						delta := cleaned[lastEmittedCleanLen:]
						lastEmittedCleanLen = len(cleaned)
						fullBuilder.WriteString(delta)
						emitBuf += delta
						if onChunk != nil && (len(emitBuf) >= emitBatchSize || strings.Contains(emitBuf, "\n")) {
							onChunk(emitBuf)
							emitBuf = ""
						}
					}
					goto next
				}
			}
			if len(raw) >= 12 || strings.Contains(lower, "\n") {
				prefixDone = true
				rest := raw
				for _, p := range []string{"assistant: ", "assistant:", "Assistant: ", "Assistant:"} {
					if strings.HasPrefix(lower, p) {
						rest = strings.TrimSpace(raw[len(p):])
						break
					}
				}
				streamBuf.Reset()
				if rest != "" {
					pendingLink.WriteString(rest)
					cleaned := stripPromoPhrase(removeLinks(pendingLink.String()))
					if len(cleaned) < lastEmittedCleanLen {
						lastEmittedCleanLen = len(cleaned)
					}
					delta := cleaned[lastEmittedCleanLen:]
					lastEmittedCleanLen = len(cleaned)
					fullBuilder.WriteString(delta)
					emitBuf += delta
					if onChunk != nil && (len(emitBuf) >= emitBatchSize || strings.Contains(emitBuf, "\n")) {
						onChunk(emitBuf)
						emitBuf = ""
					}
				}
			}
		next:
			continue
		}
		lower := strings.ToLower(raw)
		if strings.Contains(lower, "\nuser:") || strings.Contains(lower, "\nassistant:") {
			stopForward = true
			idx := strings.Index(lower, "\nuser:")
			if idx < 0 {
				idx = strings.Index(lower, "\nassistant:")
			}
			if idx > 0 {
				rest := raw[:idx]
				pendingLink.WriteString(rest)
				cleaned := stripPromoPhrase(removeLinks(pendingLink.String()))
				if len(cleaned) < lastEmittedCleanLen {
					lastEmittedCleanLen = len(cleaned)
				}
				delta := cleaned[lastEmittedCleanLen:]
				lastEmittedCleanLen = len(cleaned)
				fullBuilder.WriteString(delta)
				emitBuf += delta
				if onChunk != nil {
					if emitBuf != "" {
						onChunk(emitBuf)
						emitBuf = ""
					}
				}
			}
			streamBuf.Reset()
			continue
		}
		pendingLink.WriteString(content)
		cleaned := stripPromoPhrase(removeLinks(pendingLink.String()))
		if len(cleaned) < lastEmittedCleanLen {
			lastEmittedCleanLen = len(cleaned)
		}
		delta := cleaned[lastEmittedCleanLen:]
		lastEmittedCleanLen = len(cleaned)
		fullBuilder.WriteString(delta)
		emitBuf += delta
		if onChunk != nil && (len(emitBuf) >= emitBatchSize || strings.Contains(emitBuf, "\n")) {
			onChunk(emitBuf)
			emitBuf = ""
		}
		streamBuf.Reset()
	}
	if streamBuf.Len() > 0 && !stopForward {
		s := streamBuf.String()
		pendingLink.WriteString(s)
		cleaned := stripPromoPhrase(removeLinks(pendingLink.String()))
		if len(cleaned) < lastEmittedCleanLen {
			lastEmittedCleanLen = len(cleaned)
		}
		delta := cleaned[lastEmittedCleanLen:]
		fullBuilder.WriteString(delta)
		emitBuf += delta
	}
	if emitBuf != "" && onChunk != nil {
		onChunk(emitBuf)
	}
	return fullBuilder.String(), scanner.Err()
}
