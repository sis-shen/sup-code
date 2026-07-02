package deepseek

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
    if cfg.BaseURL == "" { cfg.BaseURL = "https://api.deepseek.com/v1" }
    if cfg.MaxTokens <= 0 { cfg.MaxTokens = 4096 }
    return &LLMClient{cfg: cfg, httpClient: &http.Client{Timeout: 60 * time.Second}}
}

func NewLLMClientWithHTTP(cfg Config, httpClient *http.Client) *LLMClient {
    if cfg.BaseURL == "" { cfg.BaseURL = "https://api.deepseek.com/v1" }
    if cfg.MaxTokens <= 0 { cfg.MaxTokens = 4096 }
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
        backoff := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
        for attempt := 0; attempt <= len(backoff); attempt++ {
            if attempt > 0 {
                select {
                case <-ctx.Done():
                    eventCh <- pkg.StreamEvent{Type: "error", Error: ctx.Err().Error()}; return
                case <-time.After(backoff[attempt-1]):
                }
            }
            err = c.doRequest(ctx, reqBody, eventCh)
            if err == nil { return }
            lastErr = err
            if !isTransient(err) { break }
            if attempt >= len(backoff) { break }
        }
        if lastErr != nil {
            eventCh <- pkg.StreamEvent{Type: "error", Error: fmt.Sprintf("failed: %v", lastErr)}
        }
    }()
    return eventCh, nil
}

func (c *LLMClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) {
    url := strings.TrimRight(c.cfg.BaseURL, "/") + "/models"
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
    resp, err := c.httpClient.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    var result struct {
        Data []struct {
            ID string `json:"id"`
        } `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil { return nil, err }
    models := make([]pkg.ModelInfo, 0)
    for _, m := range result.Data {
        models = append(models, pkg.ModelInfo{ID: m.ID, Name: m.ID, MaxTokens: 65536, SupportsVision: strings.Contains(m.ID, "vision")})
    }
    return models, nil
}

func (c *LLMClient) ProviderName() string { return "deepseek" }

func (c *LLMClient) buildRequest(systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) ([]byte, error) {
type chatMsg struct {
        Role    string `json:"role"`
        Content string `json:"content"`
    }
    req := struct {
        Model     string `json:"model"`
        Messages  []chatMsg  `json:"messages"`
        Stream    bool   `json:"stream"`
        MaxTokens int    `json:"max_tokens,omitempty"`
    }{Model: c.cfg.Model, MaxTokens: c.cfg.MaxTokens, Stream: true}
    if systemPrompt != "" {
			req.Messages = append(req.Messages, chatMsg{Role: "system", Content: systemPrompt})
    }
for _, m := range messages {
			req.Messages = append(req.Messages, chatMsg{Role: string(m.Role), Content: m.Content})
    }
    return json.Marshal(req)
}

func (c *LLMClient) doRequest(ctx context.Context, reqBody []byte, eventCh chan<- pkg.StreamEvent) error {
    url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
    if err != nil { return err }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
    resp, err := c.httpClient.Do(req)
    if err != nil { return fmt.Errorf("http: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode == 429 {
        return &pkg.ErrRetryable{Cause: fmt.Errorf("rate limited: %s", resp.Status)}
    }
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("unexpected %s: %s", resp.Status, string(body))
    }
    return c.parseSSE(ctx, resp.Body, eventCh)
}

func (c *LLMClient) parseSSE(ctx context.Context, body io.Reader, eventCh chan<- pkg.StreamEvent) error {
    scanner := bufio.NewScanner(body)
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            payload := strings.TrimPrefix(line, "data: ")
            if payload == "[DONE]" { eventCh <- pkg.StreamEvent{Type: "done"}; return nil }
            if payload == "" || payload == " " { continue }
            type chunkChoice struct {
                Delta struct {
                    Content string `json:"content,omitempty"`
                } `json:"delta"`
                FinishReason string `json:"finish_reason"`
            }
            type streamChunk struct {
                Choices []chunkChoice `json:"choices"`
            }
            var chunk streamChunk
            if err := json.Unmarshal([]byte(payload), &chunk); err != nil { continue }
            for _, choice := range chunk.Choices {
                if choice.Delta.Content != "" {
                    eventCh <- pkg.StreamEvent{Type: "text_delta", Delta: choice.Delta.Content}
                }
                if choice.FinishReason == "stop" {
                    eventCh <- pkg.StreamEvent{Type: "done"}; return nil
                }
            }
        }
    }
    eventCh <- pkg.StreamEvent{Type: "done"}
    return scanner.Err()
}

func isTransient(err error) bool {
    if _, ok := err.(*pkg.ErrRetryable); ok { return true }
    s := err.Error()
    return strings.Contains(s, "timeout") || strings.Contains(s, "empty") ||
        strings.Contains(s, "connection") || strings.Contains(s, "unexpected status 5")
}

var _ pkg.LLMClient = (*LLMClient)(nil)
