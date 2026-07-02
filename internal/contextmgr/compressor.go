package contextmgr

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/supcode/supcode/pkg"
)

const (
	defaultSegSize    = 10   // messages per segment in Map phase
	defaultKeepRecent = 20   // recent messages to preserve verbatim
	mapPromptTPL      = "Summarize the following conversation segment concisely, preserving any error messages, user preferences, and irreversible operations:\n\n"
	reducePromptTPL   = "Combine the following segment summaries into a single coherent summary. Keep all important context including:\n- Errors and failures\n- User preferences and decisions\n- Irreversible operations performed\n- Key architectural or design decisions\n\n"
)

// Compressor performs Map-Reduce context compression using an LLM.
type Compressor struct {
	llm       pkg.LLMClient
	segSize   int
	mu        sync.Mutex
}

// NewCompressor creates a new Compressor.
func NewCompressor(llm pkg.LLMClient) *Compressor {
	return &Compressor{
		llm:     llm,
		segSize: defaultSegSize,
	}
}

// CompressOldMessages compresses messages older than keepRecent using Map-Reduce.
// Returns the compressed summary string and any errors.
func (c *Compressor) CompressOldMessages(ctx context.Context, messages []pkg.Message, keepRecent int) (string, error) {
	if len(messages) <= keepRecent {
		return "", nil
	}

	oldMsgs := messages[:len(messages)-keepRecent]

	// Map phase: split into segments and summarize each
	segments := c.splitSegments(oldMsgs, c.segSize)

	type segResult struct {
		index int
		sum   string
		err   error
	}

	resultCh := make(chan segResult, len(segments))
	var wg sync.WaitGroup

	for i, seg := range segments {
		wg.Add(1)
		go func(idx int, msgs []pkg.Message) {
			defer wg.Done()
			sum, err := c.summarizeSegment(ctx, msgs)
			resultCh <- segResult{index: idx, sum: sum, err: err}
		}(i, seg)
	}

	wg.Wait()
	close(resultCh)

	// Collect results in order
	summaries := make([]string, len(segments))
	for res := range resultCh {
		if res.err != nil {
			return "", fmt.Errorf("map phase segment %d: %w", res.index, res.err)
		}
		summaries[res.index] = res.sum
	}

	if len(summaries) == 0 {
		return "", nil
	}

	// Reduce phase: if only one segment, use its summary directly
	if len(summaries) == 1 {
		return summaries[0], nil
	}

	// Otherwise, merge all summaries into one
	return c.reduceSummaries(ctx, summaries)
}

// summarizeSegment calls the LLM to summarize a single segment.
func (c *Compressor) summarizeSegment(ctx context.Context, messages []pkg.Message) (string, error) {
	content := c.formatSegment(messages)
	prompt := mapPromptTPL + content

	chatMessages := []pkg.Message{
		{Role: pkg.RoleUser, Content: prompt},
	}

	stream, err := c.llm.Chat(ctx, "You are a precise summarizer. Focus on preserving factual information, errors, and decisions.", chatMessages, nil)
	if err != nil {
		return "", err
	}

	return collectContent(stream)
}

// reduceSummaries merges multiple segment summaries into one.
func (c *Compressor) reduceSummaries(ctx context.Context, summaries []string) (string, error) {
	content := strings.Join(summaries, "\n---\n")
	prompt := reducePromptTPL + content

	chatMessages := []pkg.Message{
		{Role: pkg.RoleUser, Content: prompt},
	}

	stream, err := c.llm.Chat(ctx, "You are a precise summarizer. Merge the following conversation segment summaries into one coherent summary.", chatMessages, nil)
	if err != nil {
		return "", err
	}

	return collectContent(stream)
}

// splitSegments splits a message slice into segments of the given size.
func (c *Compressor) splitSegments(messages []pkg.Message, segSize int) [][]pkg.Message {
	if segSize <= 0 {
		segSize = defaultSegSize
	}
	var segments [][]pkg.Message
	for i := 0; i < len(messages); i += segSize {
		end := i + segSize
		if end > len(messages) {
			end = len(messages)
		}
		segments = append(segments, messages[i:end])
	}
	return segments
}

// formatSegment formats a segment of messages as a text block.
func (c *Compressor) formatSegment(messages []pkg.Message) string {
	var b strings.Builder
	for _, msg := range messages {
		b.WriteString(string(msg.Role))
		b.WriteString(": ")
		b.WriteString(msg.Content)
		b.WriteString("\n")

		for _, tc := range msg.ToolCalls {
			b.WriteString(fmt.Sprintf("  [tool call: %s(%s)]\n", tc.Name, string(tc.Params)))
		}
		if msg.ToolID != "" {
			b.WriteString(fmt.Sprintf("  [tool result for: %s]\n", msg.ToolID))
		}
	}
	return b.String()
}

// collectContent reads all text from a stream event channel.
func collectContent(stream <-chan pkg.StreamEvent) (string, error) {
	var b strings.Builder
	for evt := range stream {
		switch evt.Type {
		case "text_delta":
			b.WriteString(evt.Delta)
		case "error":
			return "", fmt.Errorf("llm error: %s", evt.Error)
		case "done":
			// Done, continue to collect full content
		}
	}
	return b.String(), nil
}
