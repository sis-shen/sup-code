package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/supcode/supcode/pkg"
)

var (
	planTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).PaddingBottom(1)
	stepDoneStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("83"))
	stepWaitStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	stepActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	planFrameStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

type PlanView struct {
	Plan      *pkg.Plan
	Visible   bool
}

func NewPlanView(plan *pkg.Plan) *PlanView {
	return &PlanView{Plan: plan, Visible: plan != nil}
}

func (pv *PlanView) RenderPlan() string {
	if pv.Plan == nil || !pv.Visible {
		return ""
	}

	var b strings.Builder
	b.WriteString(planTitleStyle.Render(fmt.Sprintf("📋 Plan: %s", pv.Plan.Goal)))
	b.WriteString("\n")

	for _, step := range pv.Plan.Steps {
		switch step.Status {
		case "completed":
			b.WriteString(stepDoneStyle.Render(fmt.Sprintf("  ✓ %s", step.Description)) + "\n")
		case "in_progress":
			b.WriteString(stepActiveStyle.Render(fmt.Sprintf("  ▶ %s", step.Description)) + "\n")
		default:
			b.WriteString(stepWaitStyle.Render(fmt.Sprintf("  ○ %s", step.Description)) + "\n")
		}
		if step.ToolHint != "" {
			b.WriteString(stepWaitStyle.Render(fmt.Sprintf("    (tool: %s)", step.ToolHint)) + "\n")
		}
	}

	return planFrameStyle.Render(b.String())
}

func RenderConfirmPrompt() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Render("[Y] Approve  [N] Re-plan  [E] Edit plan")
}

func RenderPlanWithConfirm(plan *pkg.Plan) string {
	if plan == nil {
		return ""
	}
	pv := NewPlanView(plan)
	return pv.RenderPlan() + "\n" + RenderConfirmPrompt()
}