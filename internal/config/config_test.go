package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/supcode/supcode/pkg"
)

func TestDefaultValues(t *testing.T) {
	m := NewManager()
	m.Set(ConfigKeyLLMAPIKey, "test-key")
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	tests := []struct {
		key      string
		expected any
	}{
		{ConfigKeyLLMProvider, "openai"},
		{ConfigKeyLLMModel, "gpt-4o"},
		{ConfigKeyLLMMaxTokens, 4096},
		{ConfigKeyAgentMaxIterations, 25},
		{ConfigKeyAgentPermission, "ask"},
		{ConfigKeySkillsAutoLoad, true},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := m.Get(tt.key); got != tt.expected {
				t.Errorf("Get(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestFileLoading(t *testing.T) {
	dir := t.TempDir()
	yamlContent := []byte("llm:\n  model: gpt-4o-mini\n  provider: test-provider\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), yamlContent, 0644); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	m.Set(ConfigKeyLLMAPIKey, "test-key")
	m.v.AddConfigPath(dir)
	m.v.SetConfigName("config")
	m.v.SetConfigType("yaml")
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if got := m.GetString(ConfigKeyLLMModel); got != "gpt-4o-mini" {
		t.Errorf("GetString(%q) = %q, want %q", ConfigKeyLLMModel, got, "gpt-4o-mini")
	}
	if got := m.GetString(ConfigKeyLLMProvider); got != "test-provider" {
		t.Errorf("GetString(%q) = %q, want %q", ConfigKeyLLMProvider, got, "test-provider")
	}
}

func TestEnvOverride(t *testing.T) {
	dir := t.TempDir()
	yamlContent := []byte("llm:\n  model: gpt-4o\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), yamlContent, 0644); err != nil {
		t.Fatal(err)
	}

	oldModel := os.Getenv("SUPCODE_LLM_MODEL")
	oldKey := os.Getenv("SUPCODE_LLM_API_KEY")
	os.Setenv("SUPCODE_LLM_MODEL", "gpt-4o-mini")
	os.Setenv("SUPCODE_LLM_API_KEY", "key-from-env")
	defer func() {
		os.Setenv("SUPCODE_LLM_MODEL", oldModel)
		os.Setenv("SUPCODE_LLM_API_KEY", oldKey)
	}()

	m := NewManager()
	m.v.AddConfigPath(dir)
	m.v.SetConfigName("config")
	m.v.SetConfigType("yaml")
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if got := m.GetString(ConfigKeyLLMModel); got != "gpt-4o-mini" {
		t.Errorf("GetString(%q) = %q, want %q (env should override file)", ConfigKeyLLMModel, got, "gpt-4o-mini")
	}
	if got := m.GetString(ConfigKeyLLMAPIKey); got != "key-from-env" {
		t.Errorf("GetString(%q) = %q, want %q", ConfigKeyLLMAPIKey, got, "key-from-env")
	}
}

func TestPriorityChain(t *testing.T) {
	m := NewManager()
	m.Set(ConfigKeyLLMModel, "cli-value")
	m.Set(ConfigKeyLLMAPIKey, "cli-key")

	oldModel := os.Getenv("SUPCODE_LLM_MODEL")
	os.Setenv("SUPCODE_LLM_MODEL", "env-value")
	defer os.Setenv("SUPCODE_LLM_MODEL", oldModel)

	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if got := m.GetString(ConfigKeyLLMModel); got != "cli-value" {
		t.Errorf("GetString(%q) = %q, want %q (CLI flag should win)", ConfigKeyLLMModel, got, "cli-value")
	}
}

func TestMissingAPIKey(t *testing.T) {
	m := NewManager()
	if err := m.Load(); err == nil {
		t.Fatal("expected error when API key is missing, got nil")
	} else {
		t.Logf("got expected error: %v", err)
	}
}

func TestCompileTimeInterface(t *testing.T) {
	m := NewManager()
	var _ pkg.Config = m
	_ = m
}

func TestSaveAndLoad(t *testing.T) {
	m := NewManager()
	m.Set(ConfigKeyLLMAPIKey, "test-key")

	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer os.Setenv("HOME", oldHome)
	defer os.Setenv("USERPROFILE", oldProfile)

	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	m.Set(ConfigKeyLLMModel, "saved-model")
	if err := m.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	if got := m.Get(ConfigKeyLLMModel).(string); got != "saved-model" {
		t.Errorf("Get(%q) = %v, want %q", ConfigKeyLLMModel, got, "saved-model")
	}
	if got := m.GetInt(ConfigKeyLLMMaxTokens); got != 4096 {
		t.Errorf("GetInt(%q) = %d, want %d", ConfigKeyLLMMaxTokens, got, 4096)
	}
	if got := m.GetBool(ConfigKeySkillsAutoLoad); got != true {
		t.Errorf("GetBool(%q) = %v, want true", ConfigKeySkillsAutoLoad, got)
	}

	m2 := NewManager()
	m2.Set(ConfigKeyLLMAPIKey, "test-key")
	m2.v.AddConfigPath(tmpHome + string(filepath.Separator) + ".supcode")
	m2.v.SetConfigName("config")
	m2.v.SetConfigType("yaml")
	if err := m2.Load(); err != nil {
		t.Fatalf("second Load() failed: %v", err)
	}

	if got := m2.GetString(ConfigKeyLLMModel); got != "saved-model" {
		t.Errorf("after save+reload: GetString(%q) = %q, want %q", ConfigKeyLLMModel, got, "saved-model")
	}
}

func TestConfigKeyDefaultFunc(t *testing.T) {
	v, ok := ConfigKeyDefault(ConfigKeyLLMProvider)
	if !ok {
		t.Errorf("ConfigKeyDefault(%q) returned false", ConfigKeyLLMProvider)
	}
	if v != "openai" {
		t.Errorf("ConfigKeyDefault(%q) = %v, want %q", ConfigKeyLLMProvider, v, "openai")
	}

	v, ok = ConfigKeyDefault("nonexistent.key")
	if ok {
		t.Errorf("ConfigKeyDefault(%q) should return false for unknown keys", "nonexistent.key")
	}
}

func TestViperAccessor(t *testing.T) {
	m := NewManager()
	if m.Viper() == nil {
		t.Error("Viper() should return non-nil *viper.Viper")
	}
}

func TestProjectConfigDir(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	supcodeDir := filepath.Join(cwd, ".supcode")
	if err := os.MkdirAll(supcodeDir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(supcodeDir)

	dir := projectConfigDir()
	if dir != supcodeDir {
		t.Errorf("projectConfigDir() = %q, want %q", dir, supcodeDir)
	}
}

func TestAllSettings(t *testing.T) {
	m := NewManager()
	m.Set(ConfigKeyLLMAPIKey, "test-key")
	m.Set("flat_key", "flat_value")
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	all := m.AllSettings()
	if all["flat_key"] != "flat_value" {
		t.Errorf("AllSettings()[\"flat_key\"] = %v, want %q", all["flat_key"], "flat_value")
	}
}
