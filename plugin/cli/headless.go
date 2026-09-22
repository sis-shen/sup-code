// Package cli provides the headless pkg.InteractionService used for
// non-terminal and single-shot operation. It is the replaceable alternative to
// plugin-tui: only one interaction provider may be loaded at a time.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/supcode/supcode/pkg"
)

// headlessInteraction is the non-TUI pkg.InteractionService implementation. It
// renders stream events and notifications to Out and reads user input from In,
// which makes single-shot and CI usage possible without a terminal.
type headlessInteraction struct {
	reader      *bufio.Reader
	out         io.Writer
	autoConfirm bool
	commands    pkg.CommandRegistry
}

var _ pkg.InteractionService = (*headlessInteraction)(nil)

// newHeadlessInteraction builds a headless interaction around in/out. in and
// out must be non-nil; the caller is responsible for defaulting them.
func newHeadlessInteraction(in io.Reader, out io.Writer, autoConfirm bool, commands pkg.CommandRegistry) *headlessInteraction {
	return &headlessInteraction{
		reader:      bufio.NewReader(in),
		out:         out,
		autoConfirm: autoConfirm,
		commands:    commands,
	}
}

// StreamResponse drains stream, writing text deltas and errors to Out.
func (h *headlessInteraction) StreamResponse(ctx context.Context, _ string, stream <-chan pkg.StreamEvent) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-stream:
			if !ok {
				return nil
			}
			switch ev.Type {
			case "text_delta":
				if _, err := io.WriteString(h.out, ev.Delta); err != nil {
					return err
				}
			case "error":
				if _, err := fmt.Fprintf(h.out, "error: %s\n", ev.Error); err != nil {
					return err
				}
			}
		}
	}
}

// RequestConfirmation returns the configured auto-confirm decision.
func (h *headlessInteraction) RequestConfirmation(_ context.Context, _ string, _ pkg.ConfirmPrompt) (bool, error) {
	return h.autoConfirm, nil
}

// ReadInput reads one line from In. It returns io.EOF when the input is
// exhausted before a line is read.
func (h *headlessInteraction) ReadInput(_ context.Context, _ string) (string, error) {
	line, err := h.reader.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if err != nil {
		if errors.Is(err, io.EOF) && line != "" {
			return line, nil
		}
		return line, err
	}
	return line, nil
}

// Notify writes a single "[level] message" line to Out.
func (h *headlessInteraction) Notify(_ context.Context, _ string, level pkg.NotifyLevel, message string) {
	_, _ = fmt.Fprintf(h.out, "[%s] %s\n", level, message)
}

// HandleCommand parses a leading "/name args..." input and runs the matching
// registered command. Non-slash input and unknown commands are not handled.
func (h *headlessInteraction) HandleCommand(ctx context.Context, sessionID, input string) (pkg.CommandResult, error) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return pkg.CommandResult{Handled: false}, nil
	}

	fields := strings.Fields(strings.TrimPrefix(trimmed, "/"))
	if len(fields) == 0 || h.commands == nil {
		return pkg.CommandResult{Handled: false}, nil
	}

	cmd, ok := h.commands.Get(fields[0])
	if !ok {
		return pkg.CommandResult{Handled: false}, nil
	}
	return cmd.Run(ctx, sessionID, fields[1:])
}
