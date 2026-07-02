package contextmgr

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/supcode/supcode/pkg"
)

// CardExtractor extracts structured memory cards from compressed summaries.
type CardExtractor struct {
	mu sync.Mutex
	decisionPat    *regexp.Regexp
	errorPat       *regexp.Regexp
	preferencePat  *regexp.Regexp
	sentenceSplit  *regexp.Regexp
}

// NewCardExtractor creates a new CardExtractor.
func NewCardExtractor() *CardExtractor {
	return &CardExtractor{
		decisionPat:   regexp.MustCompile(`(?i)(` + "\u51b3\u5b9a|\u51b3\u7b56|choose|decide|approved|approve|selected|chose|switched to|use\\b" + `)`),
		errorPat:      regexp.MustCompile(`(?i)(` + "\u9519\u8bef|error|failed|failure|panic|exception|crash|bug|issue|problem" + `)`),
		preferencePat: regexp.MustCompile(`(?i)(` + "\u504f\u597d|prefer|preference|like|love|hate|dislike|want|would rather|better to" + `)`),
		sentenceSplit: regexp.MustCompile(`[\x{3002}\x{ff01}\x{ff1f}.!?\n]+`),
	}
}

// ExtractCards parses a compressed summary and extracts memory cards.
func (e *CardExtractor) ExtractCards(summary string) []pkg.MemoryCard {
	if strings.TrimSpace(summary) == "" {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	var cards []pkg.MemoryCard
	now := time.Now()

	sentences := e.splitSentences(summary)

	for _, sentence := range sentences {
		s := strings.TrimSpace(sentence)
		if s == "" {
			continue
		}

		category := e.detectCategory(s)
		cards = append(cards, pkg.MemoryCard{
			ID:        uuid.New().String(),
			Content:   s,
			Category:  category,
			CreatedAt: now,
		})
	}

	cards = dedupCards(cards)
	return cards
}

// detectCategory detects the category of a sentence based on keywords.
func (e *CardExtractor) detectCategory(sentence string) string {
	lower := strings.ToLower(sentence)

	if e.errorPat.MatchString(lower) {
		return "error"
	}
	if e.decisionPat.MatchString(lower) {
		return "decision"
	}
	if e.preferencePat.MatchString(lower) {
		return "preference"
	}
	return "context"
}

// splitSentences splits text into sentences.
func (e *CardExtractor) splitSentences(text string) []string {
	parts := e.sentenceSplit.Split(text, -1)

	var sentences []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			sentences = append(sentences, s)
		}
	}
	return sentences
}

// dedupCards removes duplicate cards by content.
func dedupCards(cards []pkg.MemoryCard) []pkg.MemoryCard {
	seen := make(map[string]bool)
	var result []pkg.MemoryCard
	for _, c := range cards {
		key := strings.TrimSpace(strings.ToLower(c.Content))
		if !seen[key] {
			seen[key] = true
			result = append(result, c)
		}
	}
	return result
}

// setCards stores memory cards on a sessionContext.
func setCards(sc *sessionContext, cards []pkg.MemoryCard) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.memoryCards = cards
}

// getCards retrieves memory cards from a sessionContext.
func getCards(sc *sessionContext) []pkg.MemoryCard {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if sc.memoryCards == nil {
		return nil
	}
	result := make([]pkg.MemoryCard, len(sc.memoryCards))
	copy(result, sc.memoryCards)
	return result
}

// getSummary returns the compressed summary from a sessionContext.
func getSummary(sc *sessionContext) string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.compressedSummary
}

// setSummary sets the compressed summary on a sessionContext.
func setSummary(sc *sessionContext, summary string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.compressedSummary = summary
}

// setCompressAsyncResult stores both summary and cards in one lock.
func setCompressAsyncResult(sc *sessionContext, summary string, cards []pkg.MemoryCard) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.compressedSummary = summary
	sc.memoryCards = cards
}

// InjectCards builds the cards section for BuildContext.
func InjectCards(cards []pkg.MemoryCard) string {
	if len(cards) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n## Previous Session Memory\n\nThe following is a summary of earlier conversation:\n\n")
	for _, c := range cards {
		b.WriteString("- [")
		b.WriteString(c.Category)
		b.WriteString("] ")
		b.WriteString(c.Content)
		b.WriteString("\n")
	}
	return b.String()
}