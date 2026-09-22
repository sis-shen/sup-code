package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/supcode/supcode/pkg"
)

// helpCommand lists every registered command. It resolves the registry lazily
// so it always reflects the current command set. It is deterministic and does
// not call the model.
type helpCommand struct {
	registry pkg.CommandRegistry
}

// Name returns the command name without the leading slash.
func (c *helpCommand) Name() string { return "help" }

// Description returns a short summary for command listings.
func (c *helpCommand) Description() string { return "List available slash commands" }

// Run renders the registered command list.
func (c *helpCommand) Run(_ context.Context, _ string, _ []string) (pkg.CommandResult, error) {
	var b strings.Builder
	b.WriteString("Available commands:\n")
	for _, cmd := range c.registry.List() {
		fmt.Fprintf(&b, "  /%s - %s\n", cmd.Name(), cmd.Description())
	}
	return pkg.CommandResult{Handled: true, Message: strings.TrimRight(b.String(), "\n")}, nil
}

// versionCommand reports the configured version string. It is deterministic and
// does not call the model.
type versionCommand struct {
	version string
}

// Name returns the command name without the leading slash.
func (c *versionCommand) Name() string { return "version" }

// Description returns a short summary for command listings.
func (c *versionCommand) Description() string { return "Show the supcode version" }

// Run returns the version string as the command message.
func (c *versionCommand) Run(_ context.Context, _ string, _ []string) (pkg.CommandResult, error) {
	return pkg.CommandResult{Handled: true, Message: c.version}, nil
}
