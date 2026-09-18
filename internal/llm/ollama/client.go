package ollama

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

	"github.com/supcode/supcode/pkg"
)

type Config struct {
	BaseURL string
	Model   string
}

type LLMClient struct {
	cfg        Config
	httpClient *http.Client
}

func NewLLMClient(cfg Config) *LLMClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	return &LLMClient{cfg: cfg, httpClient: &http.Client{Timeout: 120 * time.Second}}
}

func NewLLMClientWithHTTP(cfg Config, httpClient *http.Client) *LLMClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	return &LLMClient{cfg: cfg, httpClient: httpClient}
}

func (c *LLMClient) Chat(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
	eventCh := make(chan pkg.StreamEvent, 64)
	go func() {
		defer close(eventCh)
		reqBody, err := c.buildRequest(systemPrompt, messages, tools)
		if err != nil {
			eventCh <- pkg.StreamEvent{Type: "error", Error: fmt.Sprintf("build: %v", err)}
			return
		}
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					eventCh <- pkg.StreamEvent{Type: "error", Error: ctx.Err().Error()}
					return
				case <-time.After(time.Second):
				}
			}
			err = c.doRequest(ctx, reqBody, eventCh)
			if err == nil {
				return
			}
			lastErr = err
		}
		if lastErr != nil {
			eventCh <- pkg.StreamEvent{Type: "error", Error: fmt.Sprintf("failed: %v", lastErr)}
		}
	}()
	return eventCh, nil
}

func (c *LLMClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) {
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/api/tags"
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	models := make([]pkg.ModelInfo, 0)
	for _, m := range result.Models {
		models = append(models, pkg.ModelInfo{ID: m.Name, Name: m.Name, MaxTokens: 8192, SupportsVision: strings.Contains(m.Name, "vision")})
	}
	return models, nil
}

func (c *LLMClient) ProviderName() string { return "ollama" }

func (c *LLMClient) buildRequest(systemPrompt string, messages []pkg.Message, _ []pkg.ToolSchema) ([]byte, error) {
	type chatMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	req := struct {
		Model    string    `json:"model"`
		Messages []chatMsg `json:"messages"`
		Stream   bool      `json:"stream"`
	}{Model: c.cfg.Model, Stream: true}
	if systemPrompt != "" {
		req.Messages = append(req.Messages, chatMsg{Role: "system", Content: systemPrompt})
	}
	for _, m := range messages {
		req.Messages = append(req.Messages, chatMsg{Role: string(m.Role), Content: m.Content})
	}
	return json.Marshal(req)
}

func (c *LLMClient) doRequest(ctx context.Context, reqBody []byte, eventCh chan<- pkg.StreamEvent) error {
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected %s: %s", resp.Status, string(body))
	}
	return c.parseSSE(ctx, resp.Body, eventCh)
}

func (c *LLMClient) parseSSE(_ context.Context, body io.Reader, eventCh chan<- pkg.StreamEvent) error {
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
			type choice struct {
				Delta struct {
					Content string `json:"content,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}
			type chunk struct {
				Choices []choice `json:"choices"`
			}
			var ck chunk
			if err := json.Unmarshal([]byte(payload), &ck); err != nil {
				continue
			}
			for _, ch := range ck.Choices {
				if ch.Delta.Content != "" {
					eventCh <- pkg.StreamEvent{Type: "text_delta", Delta: ch.Delta.Content}
				}
				if ch.FinishReason == "stop" || ch.FinishReason == "tool_calls" {
					eventCh <- pkg.StreamEvent{Type: "done"}
					return nil
				}
			}
		}
	}
	return scanner.Err()
}

var _ pkg.LLMClient = (*LLMClient)(nil)
