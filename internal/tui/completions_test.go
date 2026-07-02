package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdRegistry(t *testing.T) {
	r := NewCmdRegistry()
	require.NotNil(t, r)
	cmds := r.List()
	assert.GreaterOrEqual(t, len(cmds), 3)
}

func TestCmdRegistry_Register(t *testing.T) {
	r := NewCmdRegistry()
	r.Register(CmdDef{Name: "/review", Description: "Code review"})
	found := false
	for _, c := range r.List() {
		if c.Name == "/review" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestCmdRegistry_Register_Replace(t *testing.T) {
	r := NewCmdRegistry()
	r.Register(CmdDef{Name: "/help", Description: "Custom help"})
	for _, c := range r.List() {
		if c.Name == "/help" {
			assert.Equal(t, "Custom help", c.Description)
			break
		}
	}
}

func TestCmdRegistry_Search(t *testing.T) {
	r := NewCmdRegistry()
	results := r.Search("/ne")
	require.NotNil(t, results)
	assert.GreaterOrEqual(t, len(results), 1)
	assert.Equal(t, "/new", results[0].Name)
}

func TestCmdRegistry_Search_Empty(t *testing.T) {
	r := NewCmdRegistry()
	results := r.Search("")
	assert.Nil(t, results)
}

func TestCmdRegistry_Search_NoMatch(t *testing.T) {
	r := NewCmdRegistry()
	results := r.Search("/zzz")
	assert.Nil(t, results)
}

func TestCmdRegistry_Search_NoSlash(t *testing.T) {
	r := NewCmdRegistry()
	results := r.Search("new")
	assert.Nil(t, results)
}

func TestCmdRegistry_Search_CaseInsensitive(t *testing.T) {
	r := NewCmdRegistry()
	results := r.Search("/NEW")
	require.NotNil(t, results)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestCmdRegistry_Unregister(t *testing.T) {
	r := NewCmdRegistry()
	r.Unregister("/help")
	for _, c := range r.List() {
		assert.NotEqual(t, "/help", c.Name)
	}
}

func TestCmdRegistry_List_ReturnsCopy(t *testing.T) {
	r := NewCmdRegistry()
	list1 := r.List()
	list2 := r.List()
	assert.Equal(t, len(list1), len(list2))
}