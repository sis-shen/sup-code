package pkg

import "context"

// SlashCommand is a locally handled command. Commands are dispatched by the
// interaction layer without going through the model, which keeps them fast and
// deterministic (help, config, session management, ...).
type SlashCommand interface {
	// Name is the command name without the leading slash.
	Name() string
	// Description is a short human-readable summary.
	Description() string
	// Run executes the command for the given session.
	Run(ctx context.Context, sessionID string, args []string) (CommandResult, error)
}

// CommandRegistry stores the registered slash commands. It is provided by
// plugin-command and consumed by the interaction/CLI plugins.
type CommandRegistry interface {
	// Register adds a command. Registering an existing name replaces it.
	Register(cmd SlashCommand) error
	// Unregister removes a command by name.
	Unregister(name string) error
	// Get resolves a command by name.
	Get(name string) (SlashCommand, bool)
	// List returns all commands sorted by name.
	List() []SlashCommand
}
