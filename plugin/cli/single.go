package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/supcode/supcode/pkg"
)

// SingleShot runs one non-interactive LLM turn. It appends the user input to
// the context manager, builds the context, streams a completion and stores the
// assistant reply back into the context manager. It returns the collected text.
//
// The session manager argument is part of the planned wiring but unused here:
// SingleShot reads and writes context through ctxMgr only.
func SingleShot(
	ctx context.Context,
	llm pkg.LLMClient,
	ctxMgr pkg.ContextManager,
	_ pkg.SessionManager,
	sessionID, input string,
) (string, error) {
	if llm == nil {
		return "", errors.New("cli: nil LLM client")
	}
	if ctxMgr == nil {
		return "", errors.New("cli: nil context manager")
	}

	userMsg := pkg.Message{Role: pkg.RoleUser, Content: input}
	if err := ctxMgr.AppendMessage(ctx, sessionID, userMsg); err != nil {
		return "", err
	}

	system, messages, err := ctxMgr.BuildContext(ctx, sessionID)
	if err != nil {
		return "", err
	}

	stream, err := llm.Chat(ctx, system, messages, nil)
	if err != nil {
		return "", err
	}

	var text strings.Builder
	for ev := range stream {
		switch ev.Type {
		case "text_delta":
			text.WriteString(ev.Delta)
		case "error":
			return text.String(), fmt.Errorf("cli: llm stream: %s", ev.Error)
		}
	}

	assistantMsg := pkg.Message{Role: pkg.RoleAssistant, Content: text.String()}
	if err := ctxMgr.AppendMessage(ctx, sessionID, assistantMsg); err != nil {
		return text.String(), err
	}
	return text.String(), nil
}
