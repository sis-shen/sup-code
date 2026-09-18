package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

func TestNewPlanView(t *testing.T) {
	plan := &pkg.Plan{Goal: "Test goal", Steps: []pkg.PlanItem{{ID: "1", Description: "Step 1"}}}
	pv := NewPlanView(plan)
	require.NotNil(t, pv)
	assert.True(t, pv.Visible)
}

func TestPlanView_NilPlan(t *testing.T) {
	pv := NewPlanView(nil)
	rendered := pv.RenderPlan()
	assert.Empty(t, rendered)
}

func TestPlanView_RenderPlan(t *testing.T) {
	plan := &pkg.Plan{
		Goal: "Fix the bug",
		Steps: []pkg.PlanItem{
			{ID: "1", Description: "Read the code", Status: "completed"},
			{ID: "2", Description: "Find the issue", Status: "in_progress"},
			{ID: "3", Description: "Apply fix", Status: "pending", ToolHint: "EditFile"},
		},
	}
	pv := NewPlanView(plan)
	rendered := pv.RenderPlan()
	require.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "Fix the bug")
	assert.Contains(t, rendered, "Read the code")
	assert.Contains(t, rendered, "Find the issue")
	assert.Contains(t, rendered, "Apply fix")
}

func TestPlanView_VisibleFalse(t *testing.T) {
	pv := NewPlanView(&pkg.Plan{Goal: "test"})
	pv.Visible = false
	assert.Empty(t, pv.RenderPlan())
}

func TestRenderConfirmPrompt(t *testing.T) {
	prompt := RenderConfirmPrompt()
	require.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Y")
	assert.Contains(t, prompt, "N")
	assert.Contains(t, prompt, "E")
}

func TestRenderPlanWithConfirm(t *testing.T) {
	plan := &pkg.Plan{Goal: "Test", Steps: []pkg.PlanItem{{ID: "1", Description: "Do something"}}}
	rendered := RenderPlanWithConfirm(plan)
	require.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "Test")
	assert.Contains(t, rendered, "Do something")
	assert.Contains(t, rendered, "Y")
}

func TestRenderPlanWithConfirm_Nil(t *testing.T) {
	assert.Empty(t, RenderPlanWithConfirm(nil))
}

func TestPlanView_AllStepsCompleted(t *testing.T) {
	plan := &pkg.Plan{
		Goal: "Done",
		Steps: []pkg.PlanItem{
			{ID: "1", Description: "Step 1", Status: "completed"},
			{ID: "2", Description: "Step 2", Status: "completed"},
		},
	}
	pv := NewPlanView(plan)
	rendered := pv.RenderPlan()
	require.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "✓")
}
