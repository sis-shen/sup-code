package pkg

import (
	"encoding/json"
	"testing"
	"time"
)

func roundTrip[T any](t *testing.T, v T, check func(T) bool) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !check(decoded) {
		t.Errorf("round-trip check failed for %T", v)
	}
}

func TestMessageRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	msg := Message{Role: RoleUser, Content: "hello", Timestamp: now}
	roundTrip(t, msg, func(m Message) bool {
		return m.Role == RoleUser && m.Content == "hello" && m.Timestamp.Equal(now)
	})
}

func TestMessageWithToolCalls(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		ToolCalls: []ToolCall{
			{ID: "call_1", Name: "read_file", Params: json.RawMessage(`{"path":"main.go"}`)},
		},
		Timestamp: time.Now(),
	}
	roundTrip(t, msg, func(m Message) bool {
		return len(m.ToolCalls) == 1 && m.ToolCalls[0].Name == "read_file"
	})
}

func TestToolCallParams(t *testing.T) {
	tests := []struct {
		name   string
		params string
		check  func(ToolCall) bool
	}{
		{
			name:   "null params",
			params: `null`,
			check:  func(tc ToolCall) bool { return string(tc.Params) == "null" },
		},
		{
			name:   "empty object",
			params: `{}`,
			check:  func(tc ToolCall) bool { return string(tc.Params) == "{}" },
		},
		{
			name:   "valid json",
			params: `{"path":"main.go","line":10}`,
			check:  func(tc ToolCall) bool { return string(tc.Params) == `{"path":"main.go","line":10}` },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := ToolCall{ID: "1", Name: "test", Params: json.RawMessage(tt.params)}
			roundTrip(t, tc, tt.check)
		})
	}
}

func TestToolResultRoundTrip(t *testing.T) {
	tr := ToolResult{Success: true, Data: json.RawMessage(`{"content":"ok"}`)}
	roundTrip(t, tr, func(r ToolResult) bool { return r.Success })

	trErr := ToolResult{Success: false, Error: "something went wrong"}
	roundTrip(t, trErr, func(r ToolResult) bool { return !r.Success && r.Error == "something went wrong" })
}

func TestPlanRoundTrip(t *testing.T) {
	p := Plan{
		Goal: "test task",
		Steps: []PlanItem{
			{ID: "1", Description: "step 1", Status: "pending", ToolHint: "bash"},
			{ID: "2", Description: "step 2", Status: "completed"},
		},
	}
	roundTrip(t, p, func(plan Plan) bool {
		return plan.Goal == "test task" && len(plan.Steps) == 2 &&
			plan.Steps[0].ToolHint == "bash" && plan.Steps[1].Status == "completed"
	})
}

func TestSessionRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	s := Session{
		ID:    "sess_1",
		Title: "test",
		Messages: []Message{
			{Role: RoleUser, Content: "hi", Timestamp: now},
		},
		State:      StateIdle,
		TokenCount: 10,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	roundTrip(t, s, func(sess Session) bool {
		return sess.ID == "sess_1" && sess.State == StateIdle &&
			len(sess.Messages) == 1 && sess.Messages[0].Content == "hi"
	})
}

func TestDecisionAction(t *testing.T) {
	a := Action{Type: "command", Target: "ls -la"}
	roundTrip(t, a, func(act Action) bool {
		return act.Type == "command" && act.Target == "ls -la"
	})
}

func TestStreamEvent(t *testing.T) {
	se := StreamEvent{Type: "text_delta", Delta: "Hello"}
	roundTrip(t, se, func(e StreamEvent) bool { return e.Type == "text_delta" && e.Delta == "Hello" })

	seTool := StreamEvent{
		Type:     "tool_call",
		ToolCall: &ToolCall{ID: "call_1", Name: "read_file"},
	}
	roundTrip(t, seTool, func(e StreamEvent) bool { return e.ToolCall != nil && e.ToolCall.Name == "read_file" })
}

func TestAgentResult(t *testing.T) {
	ar := AgentResult{Summary: "done", Plan: &Plan{Goal: "test"}}
	roundTrip(t, ar, func(r AgentResult) bool {
		return r.Summary == "done" && r.Plan != nil && r.Plan.Goal == "test"
	})
}

func TestConfirmPrompt(t *testing.T) {
	cp := ConfirmPrompt{Title: "Confirm", ActionType: "file_write", ActionDetail: "write to main.go"}
	roundTrip(t, cp, func(p ConfirmPrompt) bool { return p.Title == "Confirm" && p.ActionType == "file_write" })
}

func TestCommandResult(t *testing.T) {
	cr := CommandResult{Handled: true, Message: "session created", NewSession: true}
	roundTrip(t, cr, func(r CommandResult) bool {
		return r.Handled && r.NewSession && r.Message == "session created"
	})
}

func TestSubTask(t *testing.T) {
	st := SubTask{ID: "t1", Description: "do thing", Context: "ctx"}
	roundTrip(t, st, func(s SubTask) bool { return s.ID == "t1" && s.Context == "ctx" })

	sr := SubTaskResult{TaskID: "t1", Success: true, Output: "done"}
	roundTrip(t, sr, func(r SubTaskResult) bool { return r.Success && r.TaskID == "t1" })
}

func TestMemoryCard(t *testing.T) {
	mc := MemoryCard{ID: "mc1", Content: "summary here", Category: "decision"}
	roundTrip(t, mc, func(c MemoryCard) bool { return c.ID == "mc1" && c.Category == "decision" })
}

func TestMemoryEntry(t *testing.T) {
	me := MemoryEntry{ID: "e1", Scope: "project", Category: "structure", Content: "project layout"}
	roundTrip(t, me, func(e MemoryEntry) bool { return e.Scope == "project" && e.Category == "structure" })
}

func TestPermissionRule(t *testing.T) {
	pr := PermissionRule{ID: "rule1", Scope: "command", Pattern: "git *", Decision: DecisionAllow, Priority: 1}
	roundTrip(t, pr, func(r PermissionRule) bool {
		return r.ID == "rule1" && r.Decision == DecisionAllow && r.Priority == 1
	})
}

func TestWorktreeInfo(t *testing.T) {
	wi := WorktreeInfo{
		AgentID: "agent1", Path: "/tmp/work", Branch: "feature/x",
		Status: "active", CreatedAt: time.Now(),
	}
	roundTrip(t, wi, func(i WorktreeInfo) bool {
		return i.AgentID == "agent1" && i.Branch == "feature/x"
	})
}

func TestMCPServerConfig(t *testing.T) {
	cfg := MCPServerConfig{
		Name: "my-server", Command: "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-filesystem"},
		Transport: "stdio",
	}
	roundTrip(t, cfg, func(c MCPServerConfig) bool {
		return c.Name == "my-server" && c.Transport == "stdio" && len(c.Args) == 2
	})
}

func TestModelInfo(t *testing.T) {
	mi := ModelInfo{ID: "gpt-4o", Name: "GPT-4o", MaxTokens: 128000, SupportsVision: true}
	roundTrip(t, mi, func(m ModelInfo) bool { return m.ID == "gpt-4o" && m.SupportsVision })
}

func TestToolSchema(t *testing.T) {
	ts := ToolSchema{Name: "read_file", Description: "Read a file", Parameters: json.RawMessage(`{"type":"object"}`)}
	roundTrip(t, ts, func(s ToolSchema) bool {
		return s.Name == "read_file" && string(s.Parameters) == `{"type":"object"}`
	})
}

func TestConstants(t *testing.T) {
	if string(RoleSystem) != "system" {
		t.Errorf("RoleSystem = %q, want %q", RoleSystem, "system")
	}
	if string(RoleUser) != "user" {
		t.Errorf("RoleUser = %q, want %q", RoleUser, "user")
	}
	if string(RoleAssistant) != "assistant" {
		t.Errorf("RoleAssistant = %q, want %q", RoleAssistant, "assistant")
	}
	if string(RoleTool) != "tool" {
		t.Errorf("RoleTool = %q, want %q", RoleTool, "tool")
	}
}

func TestLoopStateConstants(t *testing.T) {
	states := []LoopState{StateIdle, StatePlanning, StateWaitingApproval, StateActing, StateObserving, StateCompleted, StateError}
	if len(states) != 7 {
		t.Errorf("expected 7 loop states, got %d", len(states))
	}
}

func TestDecisionConstants(t *testing.T) {
	if DecisionAllow != "allow" || DecisionAsk != "ask" || DecisionDeny != "deny" {
		t.Error("decision constants mismatch")
	}
}

func TestNotifyLevelConstants(t *testing.T) {
	if NotifyInfo != "info" || NotifyWarn != "warn" || NotifyError != "error" {
		t.Error("notify level constants mismatch")
	}
}
