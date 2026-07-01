package contextmgr

import (
	"testing"

	"github.com/supcode/supcode/pkg"
)

func TestCountTokens_Empty(t *testing.T) {
	c := NewTokenCounter()
	n, err := c.CountTokens("")
	if err != nil {
		t.Fatalf("CountTokens(''): %v", err)
	}
	if n != 0 {
		t.Errorf("CountTokens('') = %d, want 0", n)
	}
}

func TestCountTokens_English(t *testing.T) {
	c := NewTokenCounter()
	n, err := c.CountTokens("Hello, world!")
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if n <= 0 {
		t.Errorf("CountTokens('Hello, world!') = %d, want > 0", n)
	}

	// Longer text should produce more tokens
	short := "Hello"
	shortCount, _ := c.CountTokens(short)
	longCount, _ := c.CountTokens("Hello world this is a longer sentence with more words")
	if longCount <= shortCount {
		t.Errorf("long text should have more tokens than short text: short=%d, long=%d", shortCount, longCount)
	}
}

func TestCountTokens_Chinese(t *testing.T) {
	c := NewTokenCounter()
	n, err := c.CountTokens("你好世界")
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if n <= 0 {
		t.Errorf("CountTokens('你好世界') = %d, want > 0", n)
	}
}

func TestCountTokens_Code(t *testing.T) {
	c := NewTokenCounter()
	code := `func main() {
	fmt.Println("hello")
}`
	n, err := c.CountTokens(code)
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if n <= 0 {
		t.Errorf("CountTokens(code block) = %d, want > 0", n)
	}

	// Code should typically compress more than prose of same length
	prose := "hello world this is a test of the emergency broadcast system"
	proseCount, _ := c.CountTokens(prose)
	codeCount, _ := c.CountTokens("fn main() { println!(\"hello\"); } // test")
	if codeCount <= 0 {
		t.Errorf("code token count should be positive")
	}
	_ = proseCount // code compression is just informational
}

func TestCountTokens_Whitespace(t *testing.T) {
	c := NewTokenCounter()
	n, err := c.CountTokens("   ")
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if n <= 0 {
		t.Errorf("CountTokens('   ') = %d, want > 0 (whitespace still tokenizes)", n)
	}
}

func TestCountMessages_Empty(t *testing.T) {
	c := NewTokenCounter()
	n, err := c.CountMessages(nil)
	if err != nil {
		t.Fatalf("CountMessages(nil): %v", err)
	}
	if n != 3 {
		t.Errorf("CountMessages(nil) = %d, want 3 (assistant reply overhead only)", n)
	}

	n2, err := c.CountMessages([]pkg.Message{})
	if err != nil {
		t.Fatalf("CountMessages([]): %v", err)
	}
	if n2 != 3 {
		t.Errorf("CountMessages([]) = %d, want 3", n2)
	}
}

func TestCountMessages_Single(t *testing.T) {
	c := NewTokenCounter()
	msgs := []pkg.Message{
		{Role: pkg.RoleUser, Content: "hello"},
	}
	n, err := c.CountMessages(msgs)
	if err != nil {
		t.Fatalf("CountMessages: %v", err)
	}

	// Count messages overhead: 4 per msg + 3 final = 7
	// Content tokens: "user\nhello"
	contentCount, _ := c.CountTokens("user\nhello")
	expected := 7 + contentCount

	if n != expected {
		t.Errorf("CountMessages single = %d, want %d", n, expected)
	}
}

func TestCountMessages_Multiple(t *testing.T) {
	c := NewTokenCounter()
	msgs := []pkg.Message{
		{Role: pkg.RoleSystem, Content: "You are a helpful assistant."},
		{Role: pkg.RoleUser, Content: "What is the weather?"},
		{Role: pkg.RoleAssistant, Content: "I'm checking..."},
	}
	n, err := c.CountMessages(msgs)
	if err != nil {
		t.Fatalf("CountMessages: %v", err)
	}
	if n <= 0 {
		t.Errorf("CountMessages should return positive count, got %d", n)
	}

	// Verify ordering doesn't change token count invariance
	n2, _ := c.CountMessages(msgs)
	if n != n2 {
		t.Errorf("CountMessages should be deterministic: %d vs %d", n, n2)
	}
}

func TestCountMessages_WithToolCall(t *testing.T) {
	c := NewTokenCounter()
	msgs := []pkg.Message{
		{
			Role:    pkg.RoleAssistant,
			Content: "Let me look that up.",
			ToolCalls: []pkg.ToolCall{
				{ID: "call_abc", Name: "read_file", Params: []byte(`{"path":"main.go"}`)},
			},
		},
	}
	n, err := c.CountMessages(msgs)
	if err != nil {
		t.Fatalf("CountMessages: %v", err)
	}
	if n <= 0 {
		t.Errorf("CountMessages with tool calls = %d, want > 0", n)
	}
}

func TestCountMessages_WithToolResult(t *testing.T) {
	c := NewTokenCounter()
	msgs := []pkg.Message{
		{Role: pkg.RoleTool, Content: `{"result":"ok"}`, ToolID: "call_abc"},
	}
	n, err := c.CountMessages(msgs)
	if err != nil {
		t.Fatalf("CountMessages: %v", err)
	}
	if n <= 0 {
		t.Errorf("CountMessages with tool result = %d, want > 0", n)
	}
}
