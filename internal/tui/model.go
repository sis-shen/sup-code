package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/supcode/supcode/pkg"
)

// ─── Messages ─────────────────────────────────────────────────

// StreamEventMsg 携带一个 StreamEvent 给 Bubble Tea 处理
type StreamEventMsg struct {
	SessionID string
	Event     pkg.StreamEvent
}

// ConfirmResultMsg 用户确认结果
type ConfirmResultMsg struct {
	SessionID string
	Approved  bool
}

// NotificationMsg 通知消息
type NotificationMsg struct {
	SessionID string
	Level     pkg.NotifyLevel
	Message   string
}

// ErrorMsg 错误消息
type ErrorMsg struct {
	SessionID string
	Err       error
}

// ─── Styles ────────────────────────────────────────────────────

var (
	styleUser = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")). // blue
			Bold(true)

	styleAssistant = lipgloss.NewStyle().
			Foreground(lipgloss.Color("83")). // green
			Bold(true)

	styleSystem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")). // gray
			Italic(true)

	styleStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")). // yellow
			PaddingLeft(1).
			PaddingRight(1)

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")). // red
			Bold(true)

	styleThinking = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")). // purple
			Italic(true)
)

// ─── View modes ────────────────────────────────────────────────

type viewMode int

const (
	modeNormal viewMode = iota
	modeConfirm
)

// ─── Model ─────────────────────────────────────────────────────

// Model 是 Bubble Tea 的主模型
type Model struct {
	// 核心组件
	service  *Service
	renderer *Renderer
	spinner  spinner.Model
	viewport viewport.Model
	input    textinput.Model

	// 会话状态
	sessionID  string
	messages   []renderedMessage
	currentMsg strings.Builder

	// 布局状态
	viewMode viewMode
	width    int
	height   int
	ready    bool

	// 确认对话框
	confirmPrompt pkg.ConfirmPrompt
	confirmDone   chan bool

	// 输入历史
	inputHistory []string
	historyIdx   int

	// Thinking 状态
	isThinking     bool
	thinkingTool   string
	thinkingParams string

	// 状态信息
	status string
	err    error
}

// renderedMessage 渲染后的消息
type renderedMessage struct {
	role    pkg.Role
	content string
}

// NewModel 创建 TUI 模型
func NewModel(service *Service, renderer *Renderer, sessionID string) *Model {
	ti := textinput.New()
	ti.Placeholder = "Type your message here... (/help for commands)"
	ti.Focus()
	ti.CharLimit = 10000
	ti.Width = 80

	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	s.Spinner = spinner.Dot

	return &Model{
		service:    service,
		renderer:   renderer,
		spinner:    s,
		input:      ti,
		sessionID:  sessionID,
		messages:   []renderedMessage{},
		status:     "Ready",
		ready:      false,
		viewMode:   modeNormal,
		historyIdx: -1,
	}
}

// Init 初始化模型
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		textinput.Blink,
		m.waitForEvents(),
	)
}

// Update 处理消息更新
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width-2, msg.Height-6)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 2
			m.viewport.Height = msg.Height - 6
		}

		m.input.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case StreamEventMsg:
		return m.handleStreamEvent(msg)

	case NotificationMsg:
		m.status = fmt.Sprintf("[%s] %s", msg.Level, msg.Message)
		return m, nil

	case ConfirmResultMsg:
		m.viewMode = modeNormal
		m.status = "Ready"
		if msg.Approved {
			m.addMessage(pkg.RoleSystem, "✓ Confirmed")
		} else {
			m.addMessage(pkg.RoleSystem, "✗ Confirmation rejected")
		}

		if m.confirmDone != nil {
			m.confirmDone <- msg.Approved
		}
		return m, nil

	case ErrorMsg:
		m.err = msg.Err
		m.status = fmt.Sprintf("Error: %v", msg.Err)
		return m, nil

	case agentResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.err)
			m.err = msg.err
			m.isThinking = false
		} else if msg.result != nil {
			if msg.result.Summary != "" {
				m.addMessage(pkg.RoleAssistant, msg.result.Summary)
			}
			m.isThinking = false
			m.status = "Ready"
		}
		return m, nil

	case spinner.TickMsg:
		m.spinner, _ = m.spinner.Update(msg)
		return m, nil

	default:
		return m, nil
	}
}

// handleKeyMsg 处理键盘消息
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case modeConfirm:
		return m.handleConfirmKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

// handleNormalKey 处理普通模式的键盘输入
func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEnter:
		input := strings.TrimSpace(m.input.Value())
		if input == "" {
			return m, nil
		}

		// 处理斜杠命令
		if strings.HasPrefix(input, "/") {
			cmdResult, err := m.service.HandleCommand(context.Background(), m.sessionID, input)
			if err == nil && cmdResult.Handled {
				if cmdResult.NewSession {
					m.messages = []renderedMessage{}
					m.currentMsg.Reset()
					m.viewport.SetContent("")
					m.status = "New session created"
				}
				m.addMessage(pkg.RoleSystem, cmdResult.Message)
				m.input.SetValue("")
				return m, nil
			}
		}

		// 添加用户消息
		m.addMessage(pkg.RoleUser, input)
		m.input.SetValue("")
		m.historyIdx = -1

		// 添加输入到历史
		m.inputHistory = append(m.inputHistory, input)
		if len(m.inputHistory) > 50 {
			m.inputHistory = m.inputHistory[1:]
		}

		m.isThinking = true
		m.thinkingTool = ""
		m.thinkingParams = ""
		m.status = "Thinking..."
		return m, m.service.RunQueryCmd(context.Background(), m.sessionID, input)

	case tea.KeyUp:
		if len(m.inputHistory) > 0 {
			if m.historyIdx < 0 {
				m.historyIdx = len(m.inputHistory) - 1
			} else if m.historyIdx > 0 {
				m.historyIdx--
			}
			m.input.SetValue(m.inputHistory[m.historyIdx])
		}
		return m, nil

	case tea.KeyDown:
		if m.historyIdx >= 0 && m.historyIdx < len(m.inputHistory)-1 {
			m.historyIdx++
			m.input.SetValue(m.inputHistory[m.historyIdx])
		} else {
			m.historyIdx = -1
			m.input.SetValue("")
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

// handleConfirmKey 处理确认模式的键盘输入
func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeySpace:
		input := strings.ToLower(strings.TrimSpace(m.input.Value()))
		approved := input == "y" || input == "yes"
		return m, func() tea.Msg {
			return ConfirmResultMsg{
				SessionID: m.sessionID,
				Approved:  approved,
			}
		}

	case tea.KeyEscape:
		return m, func() tea.Msg {
			return ConfirmResultMsg{
				SessionID: m.sessionID,
				Approved:  false,
			}
		}

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

// handleStreamEvent 处理流式事件
func (m *Model) handleStreamEvent(msg StreamEventMsg) (tea.Model, tea.Cmd) {
	switch msg.Event.Type {
	case "text_delta":
		m.currentMsg.WriteString(msg.Event.Delta)
		m.isThinking = false
		m.updateMessageContent()

	case "tool_call":
		if msg.Event.ToolCall != nil {
			m.isThinking = true
			m.thinkingTool = msg.Event.ToolCall.Name
			if len(msg.Event.ToolCall.Params) > 80 {
				m.thinkingParams = string(msg.Event.ToolCall.Params[:80]) + "..."
			} else {
				m.thinkingParams = string(msg.Event.ToolCall.Params)
			}
			m.status = fmt.Sprintf("Using tool: %s", m.thinkingTool)
		}

	case "done":
		if m.currentMsg.Len() > 0 {
			content := m.currentMsg.String()
			rendered, err := m.renderer.RenderMarkdown(content)
			if err != nil {
				rendered = content
			}
			m.messages = append(m.messages, renderedMessage{
				role:    pkg.RoleAssistant,
				content: rendered,
			})
			m.currentMsg.Reset()
		}
		m.isThinking = false
		m.status = "Ready"

	case "error":
		m.isThinking = false
		m.status = fmt.Sprintf("Error: %s", msg.Event.Error)
		m.err = fmt.Errorf("%s", msg.Event.Error)
	}

	m.updateViewport()
	return m, nil
}

// waitForEvents 等待事件的命令
func (m *Model) waitForEvents() tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

// addMessage 添加消息到列表
func (m *Model) addMessage(role pkg.Role, content string) {
	m.messages = append(m.messages, renderedMessage{role: role, content: content})
	m.updateViewport()
}

// updateMessageContent 更新当前正在流式渲染的消息
func (m *Model) updateMessageContent() {
	if m.currentMsg.Len() == 0 {
		return
	}

	content := m.currentMsg.String()
	rendered, err := m.renderer.RenderMarkdown(content)
	if err != nil {
		rendered = content
	}

	if len(m.messages) > 0 && m.messages[len(m.messages)-1].role == pkg.RoleAssistant {
		m.messages[len(m.messages)-1].content = rendered
	} else {
		m.messages = append(m.messages, renderedMessage{
			role:    pkg.RoleAssistant,
			content: rendered,
		})
	}

	m.updateViewport()
}

// updateViewport 更新视口内容
func (m *Model) updateViewport() {
	var b strings.Builder

	for _, msg := range m.messages {
		switch msg.role {
		case pkg.RoleUser:
			b.WriteString(styleUser.Render("You:") + "\n")
			b.WriteString(msg.content + "\n\n")
		case pkg.RoleAssistant:
			b.WriteString(styleAssistant.Render("Assistant:") + "\n")
			b.WriteString(msg.content + "\n\n")
		case pkg.RoleSystem:
			b.WriteString(styleSystem.Render(msg.content) + "\n\n")
		}
	}

	// 追加 thinking 状态
	if m.isThinking {
		b.WriteString(styleThinking.Render(
			fmt.Sprintf("%s Thinking... (%s) %s",
				m.spinner.View(), m.thinkingTool, m.thinkingParams)) + "\n")
	}

	if b.Len() > 0 {
		m.viewport.SetContent(b.String())
		m.viewport.GotoBottom()
	}
}

// RequestConfirm 显示确认对话框
func (m *Model) RequestConfirm(prompt pkg.ConfirmPrompt) <-chan bool {
	m.viewMode = modeConfirm
	m.confirmPrompt = prompt
	m.confirmDone = make(chan bool, 1)
	m.status = fmt.Sprintf("Confirm: %s (Y/n)", prompt.Title)
	m.input.SetValue("")
	m.input.Placeholder = "y/n"

	// 添加确认提示到消息列表
	confirmMsg := fmt.Sprintf("%s\n%s\n%s: %s",
		prompt.Title,
		prompt.Message,
		prompt.ActionType,
		prompt.ActionDetail)
	m.addMessage(pkg.RoleSystem, confirmMsg)

	return m.confirmDone
}

// View 渲染视图
func (m *Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var b strings.Builder

	// 消息列表
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// 输入框和状态栏
	if m.viewMode == modeConfirm {
		b.WriteString(styleStatus.Render("Confirm (Y/n): ") + m.input.View() + "\n")
	} else {
		b.WriteString(m.input.View() + "\n")
	}

	// 状态栏
	statusText := m.status
	if m.err != nil {
		statusText = styleError.Render(fmt.Sprintf("Error: %v", m.err))
	}
	b.WriteString(lipgloss.NewStyle().
		Width(m.width - 2).
		Height(1).
		Foreground(lipgloss.Color("240")).
		Render(statusText))

	return b.String()
}

// RunTUI 启动 TUI 主循环
func RunTUI(service *Service, renderer *Renderer, sessionID string) error {
	model := NewModel(service, renderer, sessionID)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	_, err := p.Run()
	return err
}

// SessionID 返回当前会话 ID
func (m *Model) SessionID() string {
	return m.sessionID
}

// Messages 返回消息列表
func (m *Model) Messages() []renderedMessage {
	return m.messages
}
