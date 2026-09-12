package internal

import (
	"fmt"

	"github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/internal/tui"
	"github.com/supcode/supcode/pkg"
)

// SupCode 是 SupCode 应用的结构体。
type SupCode struct {
	Config     pkg.Config
	LLMClient  pkg.LLMClient
	SessionMgr pkg.SessionManager
	Service    *tui.Service
	Agent      pkg.Agent
}

// NewSupCode 创建 SupCode 应用实例。
func NewSupCode(configPath string) (*SupCode, error) {
	cm := config.NewManager()
	if configPath != "" {
		cm.Viper().SetConfigFile(configPath)
	}
	if err := cm.Load(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return &SupCode{Config: cm}, nil
}

// Close 释放 SupCode 应用资源。
func (s *SupCode) Close() {
	// SessionMgr 和 Service 由 infra agent 后续填充关闭逻辑
}
