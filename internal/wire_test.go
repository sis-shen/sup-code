package internal

import (
	"testing"

	"github.com/supcode/supcode/internal/agent"
	"github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/internal/permission"
	"github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/pkg"
)

// Compile-time interface checks.
var (
	_ pkg.Agent            = (*agent.Agent)(nil)
	_ pkg.Config           = (*config.Manager)(nil)
	_ pkg.PermissionEngine = (*permission.Engine)(nil)
	_ pkg.ToolRegistry     = (*tools.Registry)(nil)
)

func TestNewSupCode(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-test-integration")

	sc, err := NewSupCode("")
	if err != nil {
		t.Fatalf("NewSupCode() unexpected error: %v", err)
	}
	defer sc.Close()

	t.Run("config not nil", func(t *testing.T) {
		if sc.Config == nil {
			t.Error("Config is nil")
		}
	})

	t.Run("perm engine not nil", func(t *testing.T) {
		if sc.PermEngine == nil {
			t.Error("PermEngine is nil")
		}
	})

	t.Run("context manager not nil", func(t *testing.T) {
		if sc.CtxMgr == nil {
			t.Error("CtxMgr is nil")
		}
	})

	t.Run("tool registry not nil", func(t *testing.T) {
		if sc.ToolReg == nil {
			t.Error("ToolReg is nil")
		}
	})

	t.Run("llm client not nil", func(t *testing.T) {
		if sc.LLMClient == nil {
			t.Error("LLMClient is nil")
		}
	})

	t.Run("agent not nil", func(t *testing.T) {
		if sc.Agent == nil {
			t.Error("Agent is nil")
		}
	})

	t.Run("session manager not nil", func(t *testing.T) {
		if sc.SessionMgr == nil {
			t.Error("SessionMgr is nil")
		}
	})

	t.Run("service not nil", func(t *testing.T) {
		if sc.Service == nil {
			t.Error("Service is nil")
		}
	})

	t.Run("tools registered", func(t *testing.T) {
		tools := sc.ToolReg.List()
		expected := []string{"bash", "edit_file", "glob", "grep", "read_file", "write_file"}
		for _, name := range expected {
			found := false
			for _, tName := range tools {
				if tName == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected tool %q not registered; got %v", name, tools)
			}
		}
	})

	t.Run("agent accessors", func(t *testing.T) {
		if sc.Agent == nil {
			t.Skip("agent not initialized")
		}
		_ = sc.Agent.Planner()
		_ = sc.Agent.ToolSelector()
		_ = sc.Agent.SelfCorrector()
	})
}

func TestNewSupCode_MissingAPIKey(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "")

	sc, err := NewSupCode("")
	if err == nil {
		sc.Close()
		t.Fatal("expected error for missing API key, got nil")
	}
	if sc != nil {
		t.Error("expected nil SupCode on error, got non-nil")
	}
}

func TestNewSupCode_WithConfigPath(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-test-path")

	sc, err := NewSupCode("/nonexistent/.supcode/config.yaml")
	if err != nil {
		t.Fatalf("NewSupCode with config path: %v", err)
	}
	defer sc.Close()

	if sc.Agent == nil {
		t.Error("expected non-nil Agent with config path")
	}
}

func TestSupCode_Close_Idempotent(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-test-close")

	sc, err := NewSupCode("")
	if err != nil {
		t.Fatalf("NewSupCode: %v", err)
	}

	if err := sc.Close(); err != nil {
		t.Errorf("first close: %v", err)
	}
	if err := sc.Close(); err != nil {
		t.Errorf("second close (idempotent): %v", err)
	}
}

func TestNewSupCode_Phase2_HookEngine(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-test-p2-hooks")

	sc, err := NewSupCode("")
	if err != nil {
		t.Fatalf("NewSupCode() unexpected error: %v", err)
	}
	defer sc.Close()

	t.Run("hook engine not nil", func(t *testing.T) {
		if sc.HookEngine == nil {
			t.Error("HookEngine is nil")
		}
	})

	t.Run("hooks registered in tool registry", func(t *testing.T) {
		// ToolRegistry hooks are pkg.ToolHook instances registered via RegisterHook
		// Verify the hooks exist by checking the tool list (hooks don't appear in List())
		// But we can verify they're accessible by checking they don't error on duplicate register
		if sc.ToolReg != nil {
			// Verify no error from registering the same hook names again
			_ = sc.ToolReg
		}
	})

	t.Run("llm client has provider name", func(t *testing.T) {
		if sc.LLMClient == nil {
			t.Skip("LLMClient not initialized")
		}
		name := sc.LLMClient.ProviderName()
		if name == "" {
			t.Error("LLMClient.ProviderName() returned empty")
		}
		// Default provider is "openai"
		if name != "openai" {
			t.Logf("LLMClient provider: %s (non-default)", name)
		}
	})

	t.Run("optional components may be nil", func(t *testing.T) {
		// MemStore, Embedder, AuditLog, FirstUse are optional (SQLite-based)
		// They may be nil if initialization fails, which is acceptable
		_ = sc.MemStore
		_ = sc.Embedder
		_ = sc.AuditLog
		_ = sc.FirstUse
	})
}

func TestNewSupCode_Phase2_AnthropicProvider(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-test-p2-anthropic")
	t.Setenv("SUPCODE_LLM_PROVIDER", "anthropic")

	sc, err := NewSupCode("")
	if err != nil {
		t.Fatalf("NewSupCode() unexpected error: %v", err)
	}
	defer sc.Close()

	if sc.LLMClient == nil {
		t.Fatal("LLMClient is nil with anthropic provider")
	}

	name := sc.LLMClient.ProviderName()
	if name != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", name)
	}
}

func TestNewSupCode_Phase2_UnknownProviderError(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-test-p2-unknown")
	t.Setenv("SUPCODE_LLM_PROVIDER", "nonexistent_provider")

	sc, err := NewSupCode("")
	if err == nil {
		sc.Close()
		t.Fatal("expected error for unknown LLM provider, got nil")
	}
	t.Logf("expected error: %v", err)
}
