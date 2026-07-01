package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

func TestModel_RequestConfirm(t *testing.T) {
	m := setupTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	prompt := pkg.ConfirmPrompt{
		Title:        "Confirm Test",
		Message:      "Is this OK?",
		ActionType:   "file_write",
		ActionDetail: "Write to test.txt",
	}

	ch := m.RequestConfirm(prompt)
	require.NotNil(t, ch)
	assert.Equal(t, modeConfirm, m.viewMode)
	assert.NotNil(t, m.confirmDone)
	assert.Contains(t, m.status, "Confirm")
}

func TestModel_RequestConfirm_AddsSystemMessage(t *testing.T) {
	m := setupTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	_ = m.RequestConfirm(pkg.ConfirmPrompt{
		Title:   "Test",
		Message: "Allow this action?",
	})

	assert.Len(t, m.Messages(), 1)
}

func TestModel_RequestConfirm_Update(t *testing.T) {
	m := setupTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	_ = m.RequestConfirm(pkg.ConfirmPrompt{Title: "Approval", Message: "?"})

	// Send ConfirmResultMsg directly to Update
	result, cmd := m.Update(ConfirmResultMsg{
		SessionID: m.sessionID,
		Approved:  true,
	})
	require.NotNil(t, result)
	assert.Equal(t, modeNormal, m.viewMode)
	_ = cmd
}