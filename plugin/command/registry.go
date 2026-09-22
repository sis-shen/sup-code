package command

import (
	"errors"
	"sort"
	"sync"

	"github.com/supcode/supcode/pkg"
)

// registry is the in-memory pkg.CommandRegistry implementation. It is safe for
// concurrent use and replaces any command registered under an existing name.
type registry struct {
	mu   sync.RWMutex
	cmds map[string]pkg.SlashCommand
}

// newRegistry returns an empty command registry.
func newRegistry() *registry {
	return &registry{cmds: make(map[string]pkg.SlashCommand)}
}

// Register adds cmd, replacing any command with the same name.
func (r *registry) Register(cmd pkg.SlashCommand) error {
	if cmd == nil {
		return errors.New("command: nil command")
	}
	name := cmd.Name()
	if name == "" {
		return errors.New("command: command name must not be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmds[name] = cmd
	return nil
}

// Unregister removes the command named name. Removing an unknown command is a
// no-op so callers can unregister unconditionally.
func (r *registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cmds, name)
	return nil
}

// Get resolves a command by name.
func (r *registry) Get(name string) (pkg.SlashCommand, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.cmds[name]
	return cmd, ok
}

// List returns all registered commands sorted by name.
func (r *registry) List() []pkg.SlashCommand {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]pkg.SlashCommand, 0, len(r.cmds))
	for _, cmd := range r.cmds {
		out = append(out, cmd)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

var _ pkg.CommandRegistry = (*registry)(nil)
