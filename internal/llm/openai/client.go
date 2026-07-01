package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/supcode/supcode/pkg"
)

type Config struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
}

type LLMClient struct {
	cfg        Config
	httpClient *http.Client
}

func NewLLMClient(cfg Config) *LLMClient {
	return &LLMClient{
		cfg: cfg,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func NewLLMClientWithHTTP(cfg Config, httpClient *http.Client) *LLMClient {
	return &LLMClient{cfg: cfg, httpClient: httpClient}
}

func (c *LLMClient) Chat(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
	eventCh := make(chan pkg.StreamEvent, 64)
	go func() {
		defer close(eventCh)
		reqBody, err := c.buildChatRequest(systemPrompt, messages, tools)
		if err != nil {
			eventCh <- pkg.StreamEvent{Type: "error", Error: fmt.Sprintf("build request: %v", err)}
			return
		}
		var lastErr error
		backoff := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
		for attempt := 0; attempt <= len(backoff); attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					eventCh <- pkg.StreamEvent{Type: "error", Error: ctx.Err().Error()}
					return
				case <-time.After(backoff[attempt-1]):
				}
			}
			err = c.doChatRequest(ctx, reqBody, eventCh)
			if err == nil {
				return
			}
			lastErr = err
			if !c.isRetryableError(err) {
				break
			}
			if retryable, ok := err.(*pkg.ErrRetryable); ok && retryable.RetryAfter > 0 && attempt < len(backoff) {
				select {
				case <-ctx.Done():
					eventCh <- pkg.StreamEvent{Type: "error", Error: ctx.Err().Error()}
					return
				case <-time.After(retryable.RetryAfter):
					continue
				}
			}
			if attempt >= len(backoff) {
				break
			}
		}
		if lastErr != nil {
			eventCh <- pkg.StreamEvent{Type: "error", Error: fmt.Sprintf("llm request failed after retries: %v", lastErr)}
		}
	}()
	return eventCh, nil
}

func (c *LLMClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) {
	baseURL := c.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	url := fmt.Sprintf("%s/v1/models", strings.TrimRight(baseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models: %s", resp.Status)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	models := make([]pkg.ModelInfo, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, pkg.ModelInfo{
			ID: m.ID, Name: m.ID, MaxTokens: 4096,
			SupportsVision: strings.Contains(m.ID, "gpt-4") || strings.Contains(m.ID, "gpt-4o"),
		})
	}
	return models, nil
}

func (c *LLMClient) ProviderName() string {
	return "openai"
}

func (c *LLMClient) buildChatRequest(systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) ([]byte, error) {
	req := openAIChatRequest{
		Model: c.cfg.Model, MaxTokens: c.cfg.MaxTokens,
		Stream: true, Temperature: 0.7,
	}
	if systemPrompt != "" {
		req.Messages = append(req.Messages, openAIMessage{Role: "system", Content: systemPrompt})
	}
	for _, msg := range messages {
		m := openAIMessage{Role: string(msg.Role), Content: msg.Content}
		if len(msg.ToolCalls) > 0 {
			m.ToolCalls = make([]openAIToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				args := tc.Params
				if args == nil {
					args = json.RawMessage("{}")
				}
				m.ToolCalls = append(m.ToolCalls, openAIToolCall{
					ID: tc.ID, Type: "function",
					Function: openAIFunctionCall{Name: tc.Name, Arguments: string(args)},
				})
			}
		}
		if msg.ToolID != "" {
			m.ToolCallID = msg.ToolID
		}
		req.Messages = append(req.Messages, m)
	}
	if len(tools) > 0 {
		req.Tools = make([]openAITool, 0, len(tools))
		for _, t := range tools {
			req.Tools = append(req.Tools, openAITool{
				Type: "function",
				Function: openAIFunctionDef{Name: t.Name, Description: t.Description, Parameters: t.Parameters},
			})
		}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return body, nil
}

func (c *LLMClient) doChatRequest(ctx context.Context, reqBody []byte, eventCh chan<- pkg.StreamEvent) error {
	baseURL := c.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	url := fmt.Sprintf("%s/v1/chat/completions", strings.TrimRight(baseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &pkg.ErrRetryable{Cause: fmt.Errorf("rate limited: %s", resp.Status), RetryAfter: retryAfter}
	}
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %s: %s", resp.Status, string(bodyBytes))
	}
	return c.parseSSEStream(ctx, resp.Body, eventCh)
}

//Fixed SSE parser: process delta content before finish_reason, return after done
func (c *LLMClient) parseSSEStream(ctx context.Context, body io.Reader, eventCh chan<- pkg.StreamEvent) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				eventCh <- pkg.StreamEvent{Type: "done"}
				return nil
			}
			var chunk openAIChatChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				eventCh <- pkg.StreamEvent{Type: "error", Error: fmt.Sprintf("parse chunk: %v", err)}
				continue
			}
			for _, choice := range chunk.Choices {
				// Process content text delta first
				if choice.Delta.Content != "" {
					eventCh <- pkg.StreamEvent{Type: "text_delta", Delta: choice.Delta.Content}
				}
				// Process tool calls
				for _, tc := range choice.Delta.ToolCalls {
					args := tc.Function.Arguments
					if args == "" {
						args = "{}"
					}
					eventCh <- pkg.StreamEvent{
						Type: "tool_call",
						ToolCall: &pkg.ToolCall{ID: tc.ID, Name: tc.Function.Name, Params: json.RawMessage(args)},
					}
				}
				// Then check finish reason and return to prevent double-done
				if choice.FinishReason != "" {
					if choice.FinishReason == "stop" || choice.FinishReason == "tool_calls" {
						eventCh <- pkg.StreamEvent{Type: "done"}
					} else if choice.FinishReason == "length" {
						eventCh <- pkg.StreamEvent{Type: "error", Error: "response exceeded max tokens"}
					}
					return nil
				}
			}
		}
	}
	return scanner.Err()
}

func (c *LLMClient) isRetryableError(err error) bool {
	if _, ok := err.(*pkg.ErrRetryable); ok {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "unexpected status 5")
}

func parseRetryAfter(val string) time.Duration {
	if val == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(val); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if t, err := time.Parse(time.RFC1123, val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
		return 0
	}
	return 0
}

var _ pkg.LLMClient = (*LLMClient)(nil)

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Tools       []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAITool struct {
	Type     string           `json:"type"`
	Function openAIFunctionDef `json:"function"`
}

type openAIFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type openAIChatChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   *openAIUsage   `json:"usage,omitempty"`
}

type openAIChoice struct {
	Index        int        `json:"index"`
	Delta        openAIDelta `json:"delta"`
	FinishReason string     `json:"finish_reason"`
}

type openAIDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
