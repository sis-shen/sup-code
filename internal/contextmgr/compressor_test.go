package contextmgr

import (
	"context"
	"testing"

	"github.com/supcode/supcode/pkg"
)

func TestSplitSegments_Empty(t *testing.T) {
	c := &Compressor{segSize: 10}
	segs := c.splitSegments(nil, 10)
	if len(segs) != 0 {
		t.Errorf("empty messages = %d segments, want 0", len(segs))
	}
}

func TestSplitSegments_SingleSegment(t *testing.T) {
	c := &Compressor{segSize: 10}
	msgs := make([]pkg.Message, 5)
	segs := c.splitSegments(msgs, 10)
	if len(segs) != 1 {
		t.Errorf("5 msgs with segSize 10 = %d segments, want 1", len(segs))
	}
	if len(segs[0]) != 5 {
		t.Errorf("segment has %d msgs, want 5", len(segs[0]))
	}
}

func TestSplitSegments_MultiSegment(t *testing.T) {
	c := &Compressor{segSize: 10}
	msgs := make([]pkg.Message, 25)
	segs := c.splitSegments(msgs, 10)
	if len(segs) != 3 {
		t.Errorf("25 msgs with segSize 10 = %d segments, want 3", len(segs))
	}
	if len(segs[0]) != 10 || len(segs[1]) != 10 || len(segs[2]) != 5 {
		t.Errorf("segment sizes: %d %d %d, want 10 10 5",
			len(segs[0]), len(segs[1]), len(segs[2]))
	}
}

func TestSplitSegments_ZeroSegSize(t *testing.T) {
	c := &Compressor{segSize: 10}
	msgs := make([]pkg.Message, 5)
	segs := c.splitSegments(msgs, 0)
	if len(segs) != 1 {
		t.Errorf("zero segSize defaults to 10, got %d segments", len(segs))
	}
}

func TestSplitSegments_ExactMultiple(t *testing.T) {
	c := &Compressor{segSize: 5}
	msgs := make([]pkg.Message, 15)
	segs := c.splitSegments(msgs, 5)
	if len(segs) != 3 {
		t.Errorf("15 msgs with segSize 5 = %d segments, want 3", len(segs))
	}
	for i, seg := range segs {
		if len(seg) != 5 {
			t.Errorf("segment %d has %d msgs, want 5", i, len(seg))
		}
	}
}

func TestFormatSegment(t *testing.T) {
	c := &Compressor{segSize: 10}
	msgs := []pkg.Message{
		{Role: pkg.RoleUser, Content: "hello"},
		{Role: pkg.RoleAssistant, Content: "world"},
	}
	result := c.formatSegment(msgs)
	expected := "user: hello\nassistant: world\n"
	if result != expected {
		t.Errorf("formatSegment = %q, want %q", result, expected)
	}
}

func TestFormatSegment_WithToolCall(t *testing.T) {
	c := &Compressor{segSize: 10}
	msgs := []pkg.Message{
		{
			Role:    pkg.RoleAssistant,
			Content: "searching",
			ToolCalls: []pkg.ToolCall{
				{ID: "call_1", Name: "search", Params: []byte(`{"q":"test"}`)},
			},
		},
		{Role: pkg.RoleTool, Content: "results", ToolID: "call_1"},
	}
	result := c.formatSegment(msgs)
	if !contains(result, "[tool call: search") {
		t.Errorf("expected tool call marker in output, got: %s", result)
	}
	if !contains(result, "[tool result for: call_1]") {
		t.Errorf("expected tool result marker in output, got: %s", result)
	}
}

func TestCompressOldMessages_UnderThreshold(t *testing.T) {
	llm := &MockLLMClient{}
	c := NewCompressor(llm)

	result, err := c.CompressOldMessages(context.Background(), []pkg.Message{}, 20)
	if err != nil {
		t.Fatalf("CompressOldMessages with empty: %v", err)
	}
	if result != "" {
		t.Errorf("empty messages returned summary: %s", result)
	}

	msgs := make([]pkg.Message, 15)
	result, err = c.CompressOldMessages(context.Background(), msgs, 20)
	if err != nil {
		t.Fatalf("CompressOldMessages under threshold: %v", err)
	}
	if result != "" {
		t.Errorf("messages below keepRecent returned summary: %s", result)
	}
}

func TestCompressOldMessages_SingleSegment(t *testing.T) {
	called := false
	llm := &MockLLMClient{
		ChatFunc: func(_ context.Context, _ string, messages []pkg.Message, _ []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
			called = true
			ch := make(chan pkg.StreamEvent, 2)
			ch <- pkg.StreamEvent{Type: "text_delta", Delta: "summarized content"}
			ch <- pkg.StreamEvent{Type: "done"}
			close(ch)
			return ch, nil
		},
	}

	c := NewCompressor(llm)
	msgs := make([]pkg.Message, 30)
	result, err := c.CompressOldMessages(context.Background(), msgs, 20)
	if err != nil {
		t.Fatalf("CompressOldMessages: %v", err)
	}
	if !called {
		t.Errorf("LLM was not called")
	}
	if result != "summarized content" {
		t.Errorf("summary = %q, want %q", result, "summarized content")
	}
}

func TestCompressOldMessages_MultiSegment(t *testing.T) {
	callCount := 0
	llm := &MockLLMClient{
		ChatFunc: func(_ context.Context, _ string, _ []pkg.Message, _ []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
			callCount++
			// First 2 calls are map phase (10+10 msgs), 3rd is reduce
			ch := make(chan pkg.StreamEvent, 2)
			ch <- pkg.StreamEvent{Type: "text_delta", Delta: "segment"}
			ch <- pkg.StreamEvent{Type: "done"}
			close(ch)
			return ch, nil
		},
	}

	c := NewCompressor(llm)
	// 30 msgs, keep 10 recent -> 20 old -> 2 segments of 10
	msgs := make([]pkg.Message, 30)
	result, err := c.CompressOldMessages(context.Background(), msgs, 10)
	if err != nil {
		t.Fatalf("CompressOldMessages: %v", err)
	}
	// Map: 2 calls (2 segments of 10), Reduce: 1 call = 3 total
	if callCount < 2 {
		t.Errorf("expected at least 2 LLM calls, got %d", callCount)
	}
	if result == "" {
		t.Errorf("expected non-empty summary from multi-segment compression")
	}
}

func TestCompressOldMessages_LLMError(t *testing.T) {
	llm := &MockLLMClient{
		ChatFunc: func(_ context.Context, _ string, _ []pkg.Message, _ []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
			ch := make(chan pkg.StreamEvent, 1)
			ch <- pkg.StreamEvent{Type: "error", Error: "something went wrong"}
			close(ch)
			return ch, nil
		},
	}

	c := NewCompressor(llm)
	msgs := make([]pkg.Message, 30)
	_, err := c.CompressOldMessages(context.Background(), msgs, 10)
	if err == nil {
		t.Errorf("expected error from LLM error event")
	}
}

func TestCollectContent(t *testing.T) {
	ch := make(chan pkg.StreamEvent, 3)
	ch <- pkg.StreamEvent{Type: "text_delta", Delta: "Hello"}
	ch <- pkg.StreamEvent{Type: "text_delta", Delta: " world"}
	ch <- pkg.StreamEvent{Type: "done"}
	close(ch)

	result, err := collectContent(ch)
	if err != nil {
		t.Fatalf("collectContent: %v", err)
	}
	if result != "Hello world" {
		t.Errorf("content = %q, want %q", result, "Hello world")
	}
}

func TestCollectContent_Error(t *testing.T) {
	ch := make(chan pkg.StreamEvent, 1)
	ch <- pkg.StreamEvent{Type: "error", Error: "api error"}
	close(ch)

	_, err := collectContent(ch)
	if err == nil {
		t.Errorf("expected error from error event")
	}
}

// MockLLMClient for testing compressor
type MockLLMClient struct {
	ChatFunc   func(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error)
	ModelsFunc func(ctx context.Context) ([]pkg.ModelInfo, error)
	ProvFunc   func() string
}

func (m *MockLLMClient) Chat(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, systemPrompt, messages, tools)
	}
	ch := make(chan pkg.StreamEvent, 1)
	ch <- pkg.StreamEvent{Type: "done"}
	close(ch)
	return ch, nil
}

func (m *MockLLMClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) {
	if m.ModelsFunc != nil {
		return m.ModelsFunc(ctx)
	}
	return nil, nil
}

func (m *MockLLMClient) ProviderName() string {
	if m.ProvFunc != nil {
		return m.ProvFunc()
	}
	return "mock"
}

var _ pkg.LLMClient = (*MockLLMClient)(nil)

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
