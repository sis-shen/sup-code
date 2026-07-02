package anthropic

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
    APIKey  string
    BaseURL string
    Model   string
}

type LLMClient struct {
    cfg        Config
    httpClient *http.Client
}

func NewLLMClient(cfg Config) *LLMClient {
    if cfg.BaseURL == "" {
        cfg.BaseURL = "https://api.anthropic.com"
    }
    return &LLMClient{cfg: cfg, httpClient: &http.Client{Timeout: 60 * time.Second}}
}

func NewLLMClientWithHTTP(cfg Config, httpClient *http.Client) *LLMClient {
    if cfg.BaseURL == "" {
        cfg.BaseURL = "https://api.anthropic.com"
    }
    return &LLMClient{cfg: cfg, httpClient: httpClient}
}

func (c *LLMClient) Chat(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
    eventCh := make(chan pkg.StreamEvent, 64)
    go func() {
        defer close(eventCh)
        reqBody, err := c.buildRequest(systemPrompt, messages, tools)
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
            err = c.doRequest(ctx, reqBody, eventCh)
            if err == nil {
                return
            }
            lastErr = err
            if !isTransient(err) { break }
            if retryable, ok := err.(*pkg.ErrRetryable); ok && retryable.RetryAfter > 0 && attempt < len(backoff) {
                select {
                case <-ctx.Done():
                    eventCh <- pkg.StreamEvent{Type: "error", Error: ctx.Err().Error()}
                    return
                case <-time.After(retryable.RetryAfter):
                    continue
                }
            }
            if attempt >= len(backoff) { break }
        }
        if lastErr != nil {
            eventCh <- pkg.StreamEvent{Type: "error", Error: fmt.Sprintf("request failed: %v", lastErr)}
        }
    }()
    return eventCh, nil
}

func (c *LLMClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) {
    return []pkg.ModelInfo{
        {ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", MaxTokens: 8192, SupportsVision: true},
        {ID: "claude-haiku-3-5-20241022", Name: "Claude Haiku 3.5", MaxTokens: 8192, SupportsVision: true},
    }, nil
}

func (c *LLMClient) ProviderName() string { return "anthropic" }

func (c *LLMClient) buildRequest(systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) ([]byte, error) {
    req := anthropicRequest{Model: c.cfg.Model, MaxTokens: 4096, Stream: true}
    if systemPrompt != "" { req.System = systemPrompt }
    for _, msg := range messages {
        am := anthropicMessage{Role: string(msg.Role), Content: []anthropicContent{}}
        if msg.Content != "" {
            if len(msg.ToolCalls) > 0 {
                am.Content = append(am.Content, anthropicContent{Type: "text", Text: msg.Content})
                for _, tc := range msg.ToolCalls {
                    args := tc.Params
                    if len(args) == 0 { args = json.RawMessage("{}") }
                    am.Content = append(am.Content, anthropicContent{
                        Type: "tool_use", ID: tc.ID, Name: tc.Name, Input: args,
                    })
                }
            } else {
                am.Content = append(am.Content, anthropicContent{Type: "text", Text: msg.Content})
            }
        }
        if msg.ToolID != "" {
            am.Content = append(am.Content, anthropicContent{
                Type: "tool_result", ToolUseID: msg.ToolID, Content: msg.Content,
            })
        }
        req.Messages = append(req.Messages, am)
    }
    for _, t := range tools {
        params := t.Parameters
        if len(params) == 0 { params = json.RawMessage(`{"type":"object"}`) }
        req.Tools = append(req.Tools, anthropicTool{
            Name: t.Name, Description: t.Description, InputSchema: params,
        })
    }
    return json.Marshal(req)
}

type anthropicRequest struct {
    Model     string              `json:"model"`
    MaxTokens int                 `json:"max_tokens"`
    Stream    bool                `json:"stream"`
    System    string              `json:"system,omitempty"`
    Messages  []anthropicMessage  `json:"messages"`
    Tools     []anthropicTool     `json:"tools,omitempty"`
}

type anthropicMessage struct {
    Role    string             `json:"role"`
    Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
    Type      string          `json:"type"`
    Text      string          `json:"text,omitempty"`
    ID        string          `json:"id,omitempty"`
    Name      string          `json:"name,omitempty"`
    Input     json.RawMessage `json:"input,omitempty"`
    ToolUseID string          `json:"tool_use_id,omitempty"`
    Content   string          `json:"content,omitempty"`
}

type anthropicTool struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicStreamChunk struct {
    Type          string                  `json:"type"`
    Index         int                     `json:"index,omitempty"`
    Delta         anthropicStreamDelta    `json:"delta,omitempty"`
    ContentBlock  *anthropicContentBlock  `json:"content_block,omitempty"`
}

type anthropicStreamDelta struct {
    Text        string `json:"text,omitempty"`
    StopReason  string `json:"stop_reason,omitempty"`
    PartialJSON string `json:"partial_json,omitempty"`
}

type anthropicContentBlock struct {
    Type  string          `json:"type"`
    ID    string          `json:"id,omitempty"`
    Name  string          `json:"name,omitempty"`
    Input json.RawMessage `json:"input,omitempty"`
    Text  string          `json:"text,omitempty"`
}

func (c *LLMClient) doRequest(ctx context.Context, reqBody []byte, eventCh chan <-pkg.StreamEvent) error {
    url := strings.TrimRight(c.cfg.BaseURL, "/") + "/v1/messages"
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
    if err != nil { return err }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", c.cfg.APIKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("Accept", "text/event-stream")
    resp, err := c.httpClient.Do(req)
    if err != nil { return fmt.Errorf("http: %w", err) }
    defer resp.Body.Close()
    if resp.StatusCode == http.StatusTooManyRequests {
        after := parseRetryAfter(resp.Header.Get("Retry-After"))
        return &pkg.ErrRetryable{Cause: fmt.Errorf("rate limited: %s", resp.Status), RetryAfter: after}
    }
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("unexpected %s: %s", resp.Status, string(body))
    }
    return c.parseSSE(ctx, resp.Body, eventCh)
}

func (c *LLMClient) parseSSE(ctx context.Context, body io.Reader, eventCh chan <-pkg.StreamEvent) error {
    scanner := bufio.NewScanner(body)
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
    var eventType string
    var buf bytes.Buffer
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "event: ") {
            eventType = strings.TrimPrefix(line, "event: ")
            continue
        }
        if strings.HasPrefix(line, "data: ") {
            buf.WriteString(strings.TrimPrefix(line, "data: "))
            continue
        }
        if line == "" && buf.Len() > 0 {
            payload := buf.String()
            buf.Reset()
            switch eventType {
            case "content_block_start":
                var blk anthropicStreamChunk
                if json.Unmarshal([]byte(payload), &blk) != nil { continue }
                if blk.ContentBlock != nil && blk.ContentBlock.Type == "tool_use" {
                    eventCh <- pkg.StreamEvent{
                        Type: "tool_call",
                        ToolCall: &pkg.ToolCall{
                            ID: blk.ContentBlock.ID, Name: blk.ContentBlock.Name, Params: blk.ContentBlock.Input,
                        },
                    }
                }
            case "content_block_delta":
                var blk anthropicStreamChunk
                if json.Unmarshal([]byte(payload), &blk) != nil { continue }
                if blk.Delta.Text != "" {
                    eventCh <- pkg.StreamEvent{Type: "text_delta", Delta: blk.Delta.Text}
                }
            case "message_delta":
                var blk anthropicStreamChunk
                if json.Unmarshal([]byte(payload), &blk) != nil { continue }
                if blk.Delta.StopReason != "" {
                    eventCh <- pkg.StreamEvent{Type: "done"}
                    return nil
                }
            case "message_stop":
                eventCh <- pkg.StreamEvent{Type: "done"}
                return nil
            }
            eventType = ""
        }
    }
    return scanner.Err()
}

func isTransient(err error) bool {
    if _, ok := err.(*pkg.ErrRetryable); ok { return true }
    s := err.Error()
    return strings.Contains(s, "timeout") || strings.Contains(s, "connection refused") ||
        strings.Contains(s, "connection reset") || strings.Contains(s, "temporary") ||
        strings.Contains(s, "unexpected status 5")
}

func parseRetryAfter(val string) time.Duration {
    if val == "" { return 0 }
    var sec int
	if _, nErr := fmt.Sscanf(val, "%d", &sec); nErr == nil { return time.Duration(sec) * time.Second }
    return 0
}

var _ pkg.LLMClient = (*LLMClient)(nil)
