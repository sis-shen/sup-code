package internal

import (
	"testing"

	"github.com/supcode/supcode/internal/agent"
	"github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/internal/permission"
	"github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/pkg"
)

var (
	_ pkg.Agent            = (*agent.Agent)(nil)
	_ pkg.Config           = (*config.Manager)(nil)
	_ pkg.PermissionEngine = (*permission.Engine)(nil)
	_ pkg.ToolRegistry     = (*tools.Registry)(nil)
)

func TestNewSupCode_ConfigNotNil(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-test")
	sc, err := NewSupCode("")
	if err != nil {
		t.Fatalf("NewSupCode() unexpected error: %v", err)
	}
	defer sc.Close()
	if sc.Config == nil {
		t.Error("Config is nil")
	}
}

func TestNewSupCode_WithoutAPIKey(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "")
	// API key validation is deferred to the Agent layer, so assembling the
	// application without a key must succeed.
	sc, err := NewSupCode("")
	if err != nil {
		t.Fatalf("NewSupCode() should succeed without an API key, got: %v", err)
	}
	defer sc.Close()
}

func TestSupCode_Close_Idempotent(t *testing.T) {
	t.Setenv("SUPCODE_LLM_API_KEY", "sk-test-close")
	sc, err := NewSupCode("")
	if err != nil {
		t.Fatalf("NewSupCode: %v", err)
	}
	sc.Close()
	sc.Close()
}
