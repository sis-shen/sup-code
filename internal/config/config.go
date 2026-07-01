package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/supcode/supcode/pkg"
)

// Manager 是 pkg.Config 接口的 Viper 实现
type Manager struct {
	v *viper.Viper
}

// NewManager 创建配置管理器
// 不立即加载，调用 Load() 才执行合并
func NewManager() *Manager {
	v := viper.New()
	v.SetEnvPrefix("SUPCODE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return &Manager{v: v}
}

// Viper 返回底层 Viper 实例，用于 Cobra flag 绑定
func (m *Manager) Viper() *viper.Viper {
	return m.v
}

// userConfigDir 返回用户级配置目录
func userConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".supcode"), nil
}

// projectConfigDir 尝试查找项目级 .supcode 目录
func projectConfigDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// 向上查找 .supcode/config.yaml
	dir := cwd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, ".supcode")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// configFileName 配置文件名（不含扩展名）
const configFileName = "config"

// Load 加载配置，合并来源：默认值 < 用户配置 < 项目配置 < 环境变量
func (m *Manager) Load() error {
	// 1. 设置默认值
	for k, v := range DefaultConfig {
		m.v.SetDefault(k, v)
	}

	// 2. 用户级配置 (~/.supcode/config.yaml)
	userDir, err := userConfigDir()
	if err == nil && userDir != "" {
		m.v.AddConfigPath(userDir)
	}

	// 3. 项目级配置 (.supcode/config.yaml，向上查找)
	if projDir := projectConfigDir(); projDir != "" {
		m.v.AddConfigPath(projDir)
	}

	// 4. 设置配置文件名并读取
	m.v.SetConfigName(configFileName)
	m.v.SetConfigType("yaml")

	// 尝试读取配置文件（不存在不报错）
	if err := m.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config: %w", err)
		}
	}

	// 5. 验证必要的配置项
	if m.v.GetString(ConfigKeyLLMAPIKey) == "" {
		return fmt.Errorf("missing required config: %s (set %s or %s env var)",
			ConfigKeyLLMAPIKey,
			"SUPCODE_LLM_API_KEY",
			"llm.api_key in config file",
		)
	}

	return nil
}

// Get 获取指定 key 的值
func (m *Manager) Get(key string) any {
	return m.v.Get(key)
}

// GetString 获取字符串配置值
func (m *Manager) GetString(key string) string {
	return m.v.GetString(key)
}

// GetInt 获取整数配置值
func (m *Manager) GetInt(key string) int {
	return m.v.GetInt(key)
}

// GetBool 获取布尔配置值
func (m *Manager) GetBool(key string) bool {
	return m.v.GetBool(key)
}

// Set 设置配置值（运行时覆盖）
func (m *Manager) Set(key string, value any) error {
	m.v.Set(key, value)
	return nil
}

// Save 持久化当前配置到用户配置目录
func (m *Manager) Save() error {
	userDir, err := userConfigDir()
	if err != nil {
		return fmt.Errorf("get user config dir: %w", err)
	}

	if err := os.MkdirAll(userDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	configPath := filepath.Join(userDir, configFileName+".yaml")
	if err := m.v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// AllSettings 返回所有配置的 map
func (m *Manager) AllSettings() map[string]any {
	return m.v.AllSettings()
}

// ── 编译期接口检查 ──────────────────────────────────────────
var _ pkg.Config = (*Manager)(nil)
