package contextmgr

import (
	"context"
	"sync"
	"testing"

	"github.com/supcode/supcode/pkg"
)

func TestManager_NewSession_TokenCountZero(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	count, err := m.TokenCount(ctx, "sess_new")
	if err != nil {
		t.Fatalf("TokenCount: %v", err)
	}
	if count != 0 {
		t.Errorf("new session TokenCount = %d, want 0", count)
	}
}

func TestManager_AppendMessage_IncreasesCount(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	initial, _ := m.TokenCount(ctx, "sess_1")
	err := m.AppendMessage(ctx, "sess_1", pkg.Message{Role: pkg.RoleUser, Content: "hello"})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	after, _ := m.TokenCount(ctx, "sess_1")
	if after <= initial {
		t.Errorf("TokenCount after AppendMessage = %d, want > %d", after, initial)
	}
}

func TestManager_BuildContext_ReturnsMessages(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	msgs := []pkg.Message{
		{Role: pkg.RoleSystem, Content: "Be helpful."},
		{Role: pkg.RoleUser, Content: "Hi there."},
	}
	for _, msg := range msgs {
		if err := m.AppendMessage(ctx, "sess_build", msg); err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
	}

	systemPrompt, messages, err := m.BuildContext(ctx, "sess_build")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if systemPrompt != "" {
		t.Errorf("BuildContext systemPrompt = %q, want empty string", systemPrompt)
	}
	if len(messages) != 2 {
		t.Fatalf("BuildContext messages count = %d, want 2", len(messages))
	}
	if messages[0].Content != "Be helpful." || messages[1].Content != "Hi there." {
		t.Errorf("BuildContext messages content mismatch: got %+v", messages)
	}
}

func TestManager_BuildContext_IsCopy(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	_ = m.AppendMessage(ctx, "sess_cpy", pkg.Message{Role: pkg.RoleUser, Content: "hello"})
	_, msgs, _ := m.BuildContext(ctx, "sess_cpy")
	if len(msgs) > 0 {
		msgs[0].Content = "mutated"
	}

	_, original, _ := m.BuildContext(ctx, "sess_cpy")
	if original[0].Content == "mutated" {
		t.Errorf("BuildContext should return a copy, not a reference")
	}
}

func TestManager_ShouldCompress_ThresholdCheck(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	sessionID := "sess_comp"
	for range 10 {
		_ = m.AppendMessage(ctx, sessionID, pkg.Message{Role: pkg.RoleUser, Content: "short"})
	}

	should, err := m.ShouldCompress(ctx, sessionID)
	if err != nil {
		t.Fatalf("ShouldCompress: %v", err)
	}
	if should {
		t.Errorf("ShouldCompress = true before threshold reached, want false")
	}

	m2 := NewManagerWithThreshold(1)
	_ = m2.AppendMessage(ctx, "sess_low", pkg.Message{Role: pkg.RoleUser, Content: "a"})
	should, _ = m2.ShouldCompress(ctx, "sess_low")
	if !should {
		t.Errorf("ShouldCompress = false with threshold=1, want true")
	}
}

func TestManager_Clear_ResetsCount(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	_ = m.AppendMessage(ctx, "sess_clr", pkg.Message{Role: pkg.RoleUser, Content: "hello"})
	if err := m.Clear(ctx, "sess_clr"); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	after, _ := m.TokenCount(ctx, "sess_clr")
	if after != 0 {
		t.Errorf("TokenCount after Clear = %d, want 0", after)
	}

	_, msgs, _ := m.BuildContext(ctx, "sess_clr")
	if len(msgs) != 0 {
		t.Errorf("messages after Clear = %d, want 0", len(msgs))
	}
}

func TestManager_ConcurrentAppend(t *testing.T) {
	m := NewManager()
	ctx := context.Background()
	sessionID := "sess_con"

	var wg sync.WaitGroup
	n := 50
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			_ = m.AppendMessage(ctx, sessionID, pkg.Message{
				Role:    pkg.RoleUser,
				Content: "hello world",
			})
		}()
	}
	wg.Wait()

	_, msgs, _ := m.BuildContext(ctx, sessionID)
	if len(msgs) != n {
		t.Errorf("message count after concurrent appends = %d, want %d", len(msgs), n)
	}
}

func TestManager_Compress_Stub(t *testing.T) {
	m := NewManager()
	ctx := context.Background()
	err := m.Compress(ctx, "sess_stub")
	if err != nil {
		t.Errorf("Compress stub should return nil, got: %v", err)
	}
}

func TestManager_GetMemoryCards_Stub(t *testing.T) {
	m := NewManager()
	ctx := context.Background()
	cards, err := m.GetMemoryCards(ctx, "sess_stub")
	if err != nil {
		t.Errorf("GetMemoryCards stub should return nil, got: %v", err)
	}
	if cards != nil {
		t.Errorf("GetMemoryCards stub should return nil cards, got: %v", cards)
	}
}

func TestManager_MultipleSessions(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	_ = m.AppendMessage(ctx, "sess_a", pkg.Message{Role: pkg.RoleUser, Content: "message for A"})
	_ = m.AppendMessage(ctx, "sess_b", pkg.Message{Role: pkg.RoleAssistant, Content: "message for B"})

	_, msgsA, _ := m.BuildContext(ctx, "sess_a")
	_, msgsB, _ := m.BuildContext(ctx, "sess_b")
	if len(msgsA) != 1 || msgsA[0].Content != "message for A" {
		t.Errorf("session A messages wrong: %+v", msgsA)
	}
	if len(msgsB) != 1 || msgsB[0].Content != "message for B" {
		t.Errorf("session B messages wrong: %+v", msgsB)
	}

	_ = m.Clear(ctx, "sess_a")
	countBafter, _ := m.TokenCount(ctx, "sess_b")
	if countBafter <= 0 {
		t.Errorf("session B token count should remain > 0 after session A clear, got %d", countBafter)
	}
}

func TestManager_AppendMessage_WithTools(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	msg := pkg.Message{
		Role:    pkg.RoleAssistant,
		Content: "Let me search.",
		ToolCalls: []pkg.ToolCall{
			{ID: "call_1", Name: "search", Params: []byte(`{"q":"weather"}`)},
		},
	}
	_ = m.AppendMessage(ctx, "sess_tool", msg)

	resultMsg := pkg.Message{
		Role:   pkg.RoleTool,
		Content: `{"result":"sunny"}`,
		ToolID: "call_1",
	}
	_ = m.AppendMessage(ctx, "sess_tool", resultMsg)

	_, msgs, _ := m.BuildContext(ctx, "sess_tool")
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestManager_AppendAfterClear(t *testing.T) {
	m := NewManager()
	ctx := context.Background()
	sessionID := "sess_reuse"

	_ = m.AppendMessage(ctx, sessionID, pkg.Message{Role: pkg.RoleUser, Content: "first"})
	_ = m.Clear(ctx, sessionID)
	_ = m.AppendMessage(ctx, sessionID, pkg.Message{Role: pkg.RoleUser, Content: "second"})

	_, msgs, _ := m.BuildContext(ctx, sessionID)
	if len(msgs) != 1 || msgs[0].Content != "second" {
		t.Errorf("expected 1 message 'second', got %+v", msgs)
	}
}

func TestManager_NewManagerWithThreshold(t *testing.T) {
	m := NewManagerWithThreshold(50)
	ctx := context.Background()

	should, _ := m.ShouldCompress(ctx, "sess_thr")
	if should {
		t.Errorf("empty session ShouldCompress = true, want false")
	}
	if m.threshold != 50 {
		t.Errorf("threshold = %d, want 50", m.threshold)
	}
}

func TestMockContextManager(t *testing.T) {
	mock := &MockContextManager{}
	ctx := context.Background()

	_, _, err := mock.BuildContext(ctx, "test")
	if err != nil {
		t.Errorf("default BuildContext should return nil error")
	}
	err = mock.AppendMessage(ctx, "test", pkg.Message{})
	if err != nil {
		t.Errorf("default AppendMessage should return nil error")
	}
	count, err := mock.TokenCount(ctx, "test")
	if err != nil || count != 0 {
		t.Errorf("default TokenCount should return 0, nil")
	}
	should, err := mock.ShouldCompress(ctx, "test")
	if err != nil || should {
		t.Errorf("default ShouldCompress should return false, nil")
	}
	err = mock.Compress(ctx, "test")
	if err != nil {
		t.Errorf("default Compress should return nil")
	}
	cards, err := mock.GetMemoryCards(ctx, "test")
	if err != nil || cards != nil {
		t.Errorf("default GetMemoryCards should return nil, nil")
	}
	err = mock.Clear(ctx, "test")
	if err != nil {
		t.Errorf("default Clear should return nil")
	}

	mock.TokenCountFunc = func(ctx context.Context, sessionID string) (int, error) {
		return 42, nil
	}
	customCount, _ := mock.TokenCount(ctx, "test")
	if customCount != 42 {
		t.Errorf("custom TokenCount = %d, want 42", customCount)
	}
}
