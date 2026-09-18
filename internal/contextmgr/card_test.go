package contextmgr

import (
	"testing"

	"github.com/supcode/supcode/pkg"
)

func TestCardExtractor_EmptySummary(t *testing.T) {
	e := NewCardExtractor()
	cards := e.ExtractCards("")
	if len(cards) != 0 {
		t.Errorf("empty summary = %d cards, want 0", len(cards))
	}

	cards = e.ExtractCards("   ")
	if len(cards) != 0 {
		t.Errorf("whitespace summary = %d cards, want 0", len(cards))
	}
}

func TestCardExtractor_DetectDecision(t *testing.T) {
	e := NewCardExtractor()
	cards := e.ExtractCards("We decided to use Go. I approve the new design.")
	if len(cards) == 0 {
		t.Fatal("expected at least one card")
	}

	hasDecision := false
	for _, c := range cards {
		if c.Category == "decision" {
			hasDecision = true
			break
		}
	}
	if !hasDecision {
		t.Errorf("expected a 'decision' card, got categories: %v", getCategories(cards))
	}
}

func TestCardExtractor_DetectError(t *testing.T) {
	e := NewCardExtractor()
	cards := e.ExtractCards("The build failed with an error. api returns 500.")
	if len(cards) == 0 {
		t.Fatal("expected at least one card")
	}

	hasError := false
	for _, c := range cards {
		if c.Category == "error" {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Errorf("expected an 'error' card, got categories: %v", getCategories(cards))
	}
}

func TestCardExtractor_DetectPreference(t *testing.T) {
	e := NewCardExtractor()
	cards := e.ExtractCards("I prefer dark mode. The user likes vim keybindings.")
	if len(cards) == 0 {
		t.Fatal("expected at least one card")
	}

	hasPreference := false
	for _, c := range cards {
		if c.Category == "preference" {
			hasPreference = true
			break
		}
	}
	if !hasPreference {
		t.Errorf("expected a 'preference' card, got categories: %v", getCategories(cards))
	}
}

func TestCardExtractor_DefaultCategory(t *testing.T) {
	e := NewCardExtractor()
	cards := e.ExtractCards("The project uses a layered architecture with three main components.")
	if len(cards) == 0 {
		t.Fatal("expected at least one card")
	}

	for _, c := range cards {
		if c.Category == "" {
			t.Errorf("card should have a non-empty category, content: %s", c.Content)
		}
		if c.ID == "" {
			t.Errorf("card should have a non-empty ID")
		}
	}
}

func TestCardExtractor_Dedup(t *testing.T) {
	e := NewCardExtractor()
	cards := e.ExtractCards("We use Go. We use Go.")
	if len(cards) != 1 {
		t.Errorf("expected 1 deduplicated card, got %d", len(cards))
	}
}

func TestCardExtractor_ChineseKeywords(t *testing.T) {
	e := NewCardExtractor()
	input := "\u51b3\u5b9a\u91c7\u7528\u5fae\u670d\u52a1\u67b6\u6784\u3002\u53d1\u751f\u4e86\u9519\u8bef\uff1a\u6570\u636e\u5e93\u8fde\u63a5\u5931\u8d25\u3002\u504f\u597d\u4f7f\u7528 PostgreSQL\u3002"
	cards := e.ExtractCards(input)
	if len(cards) == 0 {
		t.Fatal("expected cards from Chinese text")
	}

	categories := getCategories(cards)
	hasDecision := false
	hasError := false
	hasPreference := false
	for _, cat := range categories {
		switch cat {
		case "decision":
			hasDecision = true
		case "error":
			hasError = true
		case "preference":
			hasPreference = true
		}
	}

	if !hasDecision {
		t.Errorf("expected 'decision' category for Chinese '")
	}
	if !hasError {
		t.Errorf("expected 'error' category for Chinese '")
	}
	if !hasPreference {
		t.Errorf("expected 'preference' category for Chinese '")
	}
}

func TestInjectCards_Empty(t *testing.T) {
	result := InjectCards(nil)
	if result != "" {
		t.Errorf("nil cards = %q, want empty", result)
	}

	result = InjectCards([]pkg.MemoryCard{})
	if result != "" {
		t.Errorf("empty cards = %q, want empty", result)
	}
}

func TestInjectCards_FormatsCards(t *testing.T) {
	cards := []pkg.MemoryCard{
		{Category: "decision", Content: "use Go"},
		{Category: "error", Content: "build failed"},
	}
	result := InjectCards(cards)
	if result == "" {
		t.Fatal("expected non-empty injection")
	}

	if !contains(result, "[decision]") {
		t.Errorf("expected [decision] in output")
	}
	if !contains(result, "[error]") {
		t.Errorf("expected [error] in output")
	}
	if !contains(result, "use Go") {
		t.Errorf("expected card content in output")
	}
}

func TestDetectCategory(t *testing.T) {
	e := NewCardExtractor()
	tests := []struct {
		input    string
		expected string
	}{
		{"The system crashed", "error"},
		{"We decided to go with option B", "decision"},
		{"The user prefers dark mode", "preference"},
		{"Normal context information", "context"},
		{"", "context"},
	}
	for _, tt := range tests {
		got := e.detectCategory(tt.input)
		if got != tt.expected {
			t.Errorf("detectCategory(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSplitSentences(t *testing.T) {
	e := NewCardExtractor()
	text := "First sentence. Second sentence! Third sentence?\nFourth sentence."
	sentences := e.splitSentences(text)
	if len(sentences) != 4 {
		t.Errorf("expected 4 sentences, got %d: %v", len(sentences), sentences)
	}
}

func getCategories(cards []pkg.MemoryCard) []string {
	var cats []string
	for _, c := range cards {
		cats = append(cats, c.Category)
	}
	return cats
}
