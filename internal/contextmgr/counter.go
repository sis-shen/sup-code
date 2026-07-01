package contextmgr

import (
	"github.com/pkoukk/tiktoken-go"
	"github.com/supcode/supcode/pkg"
)

// TokenCounter handles token counting using OpenAI's tiktoken library.
type TokenCounter struct {
	encoding string
}

// NewTokenCounter creates a new TokenCounter using cl100k_base encoding.
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{encoding: "cl100k_base"}
}

// CountTokens returns the number of tokens in the given text using cl100k_base encoding.
func (c *TokenCounter) CountTokens(text string) (int, error) {
	tkm, err := tiktoken.GetEncoding(c.encoding)
	if err != nil {
		return 0, err
	}
	tokens := tkm.Encode(text, nil, nil)
	return len(tokens), nil
}

// CountMessages returns the total token count for a list of messages,
// including per-message format overhead (role delimiters, etc.).
func (c *TokenCounter) CountMessages(messages []pkg.Message) (int, error) {
	total := 0
	for _, msg := range messages {
		// Per-message overhead (role separator, content delimiter, etc.)
		total += 4

		count, err := c.CountTokens(string(msg.Role) + "\n" + msg.Content)
		if err != nil {
			return 0, err
		}
		total += count

		// Tool calls content
		for _, tc := range msg.ToolCalls {
			tcCount, err := c.CountTokens(tc.ID + tc.Name + string(tc.Params))
			if err != nil {
				return 0, err
			}
			total += tcCount
		}

		// Tool ID for tool-role messages
		if msg.ToolID != "" {
			tidCount, err := c.CountTokens(msg.ToolID)
			if err != nil {
				return 0, err
			}
			total += tidCount
		}
	}
	// Assistant reply overhead
	total += 3
	return total, nil
}
