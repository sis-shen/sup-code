package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleConfirmKey_EnterY(t *testing.T) {
	m := setupTestModel(t)
	m.viewMode = modeConfirm
	m.input.SetValue("y")

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	// Execute the returned command to verify it produces a ConfirmResultMsg
	msg := cmd()
	confirmMsg, ok := msg.(ConfirmResultMsg)
	require.True(t, ok, "expected ConfirmResultMsg, got %T", msg)
	assert.True(t, confirmMsg.Approved)
}

func TestHandleConfirmKey_EnterN(t *testing.T) {
	m := setupTestModel(t)
	m.viewMode = modeConfirm
	m.input.SetValue("n")

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	msg := cmd()
	confirmMsg, ok := msg.(ConfirmResultMsg)
	require.True(t, ok)
	assert.False(t, confirmMsg.Approved)
}

func TestHandleConfirmKey_Yes(t *testing.T) {
	m := setupTestModel(t)
	m.viewMode = modeConfirm
	m.input.SetValue("yes")

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	msg := cmd()
	confirmMsg, ok := msg.(ConfirmResultMsg)
	require.True(t, ok)
	assert.True(t, confirmMsg.Approved)
}

func TestHandleConfirmKey_SpaceN(t *testing.T) {
	m := setupTestModel(t)
	m.viewMode = modeConfirm
	m.input.SetValue("n")
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	require.NotNil(t, result)
	require.NotNil(t, cmd)
	_ = result
	_ = cmd
}

func TestHandleConfirmKey_Escape(t *testing.T) {
	m := setupTestModel(t)
	m.viewMode = modeConfirm

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	msg := cmd()
	confirmMsg, ok := msg.(ConfirmResultMsg)
	require.True(t, ok)
	assert.False(t, confirmMsg.Approved)
}

func TestHandleConfirmKey_Default(t *testing.T) {
	m := setupTestModel(t)
	m.viewMode = modeConfirm

	// Send a regular character key (not Enter, Space, or Escape)
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	require.NotNil(t, result)
	_ = cmd
}