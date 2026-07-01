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
