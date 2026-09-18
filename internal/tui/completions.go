package tui

import (
	"sort"
	"strings"
)

type CmdDef struct {
	Name        string
	Description string
}

type CmdRegistry struct {
	items []CmdDef
}

func NewCmdRegistry() *CmdRegistry {
	r := &CmdRegistry{}
	r.initBuiltins()
	return r
}

func (r *CmdRegistry) initBuiltins() {
	r.items = []CmdDef{
		{Name: "/new", Description: "Start a new session"},
		{Name: "/exit", Description: "Exit supcode"},
		{Name: "/help", Description: "Show this help message"},
	}
}

func (r *CmdRegistry) Register(cmd CmdDef) {
	for i, c := range r.items {
		if c.Name == cmd.Name {
			r.items[i] = cmd
			return
		}
	}
	r.items = append(r.items, cmd)
}

func (r *CmdRegistry) Unregister(name string) {
	filtered := make([]CmdDef, 0, len(r.items))
	for _, c := range r.items {
		if c.Name != name {
			filtered = append(filtered, c)
		}
	}
	r.items = filtered
}

func (r *CmdRegistry) Search(prefix string) []CmdDef {
	if prefix == "" || !strings.HasPrefix(prefix, "/") {
		return nil
	}
	prefix = strings.ToLower(prefix)
	var results []CmdDef
	for _, c := range r.items {
		if strings.HasPrefix(strings.ToLower(c.Name), prefix) {
			results = append(results, c)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})
	return results
}

func (r *CmdRegistry) List() []CmdDef {
	result := make([]CmdDef, len(r.items))
	copy(result, r.items)
	return result
}
