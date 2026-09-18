package tui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/supcode/supcode/pkg"
)

// agentResultMsg 携带 Agent.Run 结果给 Bubble Tea 处理
type agentResultMsg struct {
	sessionID string
	result    *pkg.AgentResult
	err       error
}

// ─── compile-time interface check ──────────────────────────────
var _ pkg.InteractionService = (*Service)(nil)

// Service 实现 pkg.InteractionService 接口
// 桥接引擎层事件到 Bubble Tea 消息系统
type Service struct {
	mu sync.RWMutex

	// 引擎层 Agent
	agent pkg.Agent

	// 流式响应通道映射
	streamChannels map[string]chan pkg.StreamEvent

	// 确认通道
	confirmChannels map[string]chan bool

	// 输入通道
	inputChannels map[string]chan string

	// 通知回调（由 TUI Model 注册）
	notifyHandler func(sessionID string, level pkg.NotifyLevel, message string)

	// 命令处理回调（由 TUI Model 注册）
	commandHandler func(ctx context.Context, sessionID string, input string) (pkg.CommandResult, error)

	// 会话管理器
	sessionManager *SessionManager
}

// NewService 创建交互层服务
func NewService(sm *SessionManager) *Service {
	return &Service{
		streamChannels:  make(map[string]chan pkg.StreamEvent),
		confirmChannels: make(map[string]chan bool),
		inputChannels:   make(map[string]chan string),
		sessionManager:  sm,
	}
}

// SetAgent 设置引擎层 Agent
func (s *Service) SetAgent(a pkg.Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agent = a
}

// RunQueryCmd 异步调用 Agent.Run，返回 Bubble Tea 可执行的 Cmd
func (s *Service) RunQueryCmd(ctx context.Context, sessionID, input string) tea.Cmd {
	return func() tea.Msg {
		s.mu.RLock()
		a := s.agent
		s.mu.RUnlock()

		if a == nil {
			return agentResultMsg{
				sessionID: sessionID,
				err:       fmt.Errorf("no agent configured"),
			}
		}

		result, err := a.Run(ctx, sessionID, input)
		return agentResultMsg{
			sessionID: sessionID,
			result:    result,
			err:       err,
		}
	}
}

// SetNotifyHandler 设置通知回调
func (s *Service) SetNotifyHandler(handler func(sessionID string, level pkg.NotifyLevel, message string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifyHandler = handler
}

// SetCommandHandler 设置命令处理回调
func (s *Service) SetCommandHandler(handler func(ctx context.Context, sessionID string, input string) (pkg.CommandResult, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commandHandler = handler
}

// StreamResponse 将流式事件推送给用户
func (s *Service) StreamResponse(ctx context.Context, sessionID string, stream <-chan pkg.StreamEvent) error {
	ch := make(chan pkg.StreamEvent, 100)

	s.mu.Lock()
	s.streamChannels[sessionID] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.streamChannels, sessionID)
		s.mu.Unlock()
	}()

	for {
		select {
		case evt, ok := <-stream:
			if !ok {
				close(ch)
				return nil
			}
			select {
			case ch <- evt:
			case <-ctx.Done():
				close(ch)
				return ctx.Err()
			}
		case <-ctx.Done():
			close(ch)
			return ctx.Err()
		}
	}
}

// RequestConfirmation 请求用户确认
func (s *Service) RequestConfirmation(ctx context.Context, sessionID string, prompt pkg.ConfirmPrompt) (bool, error) {
	ch := make(chan bool, 1)

	s.mu.Lock()
	s.confirmChannels[sessionID] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.confirmChannels, sessionID)
		s.mu.Unlock()
	}()

	// 发送通知给用户
	s.mu.RLock()
	if s.notifyHandler != nil {
		s.notifyHandler(sessionID, pkg.NotifyInfo,
			fmt.Sprintf("[CONFIRM] %s: %s  (Y/n)", prompt.Title, prompt.Message))
	}
	s.mu.RUnlock()

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(60 * time.Second):
		return false, nil
	}
}

// ReadInput 阻塞等待用户输入一行
func (s *Service) ReadInput(ctx context.Context, sessionID string) (string, error) {
	// 尝试通过通道获取输入（TUI 模式下由 Bubble Tea 提供）
	ch := make(chan string, 1)
	s.mu.Lock()
	s.inputChannels[sessionID] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.inputChannels, sessionID)
		s.mu.Unlock()
	}()

	// 等待输入或超时
	select {
	case input := <-ch:
		return input, nil
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("read input timeout")
	}
}

// SubmitInput 提交用户输入（由 TUI Model 调用）
func (s *Service) SubmitInput(sessionID string, input string) {
	s.mu.RLock()
	ch, ok := s.inputChannels[sessionID]
	s.mu.RUnlock()
	if ok {
		select {
		case ch <- input:
		default:
		}
	}
}

// SubmitConfirmation 提交用户确认（由 TUI Model 调用）
func (s *Service) SubmitConfirmation(sessionID string, approved bool) {
	s.mu.RLock()
	ch, ok := s.confirmChannels[sessionID]
	s.mu.RUnlock()
	if ok {
		select {
		case ch <- approved:
		default:
		}
	}
}

// Notify 向用户发送非阻塞通知
func (s *Service) Notify(ctx context.Context, sessionID string, level pkg.NotifyLevel, message string) {
	s.mu.RLock()
	handler := s.notifyHandler
	s.mu.RUnlock()

	if handler != nil {
		handler(sessionID, level, message)
	}
}

// HandleCommand 处理斜杠命令
func (s *Service) HandleCommand(ctx context.Context, sessionID string, input string) (pkg.CommandResult, error) {
	// 内置命令处理
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return pkg.CommandResult{Handled: false}, nil
	}

	cmd := strings.Fields(trimmed)
	if len(cmd) == 0 {
		return pkg.CommandResult{Handled: false}, nil
	}

	switch cmd[0] {
	case "/new":
		return pkg.CommandResult{
			Handled:    true,
			Message:    "Creating new session...",
			NewSession: true,
		}, nil
	case "/exit":
		return pkg.CommandResult{
			Handled: true,
			Message: "Goodbye!",
		}, nil
	case "/help":
		return pkg.CommandResult{
			Handled: true,
			Message: `Available commands:
  /new   - Start a new session
  /exit  - Exit supcode
  /help  - Show this help message`,
		}, nil
	default:
		// 尝试外部命令处理器
		s.mu.RLock()
		handler := s.commandHandler
		s.mu.RUnlock()

		if handler != nil {
			return handler(ctx, sessionID, input)
		}

		return pkg.CommandResult{
			Handled: false,
			Message: fmt.Sprintf("Unknown command: %s", cmd[0]),
		}, nil
	}
}

// TerminalReadInput 在非 TUI 模式下直接从终端读取输入
func TerminalReadInput(ctx context.Context) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	inputCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		if scanner.Scan() {
			inputCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	select {
	case input := <-inputCh:
		return input, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// SessionManager 返回会话管理器
func (s *Service) SessionManager() *SessionManager {
	return s.sessionManager
}
