package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type CustomCommandsConfig struct {
	Commands []CustomCommand `yaml:"commands"`
}

type CustomCommand struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Prompt      string   `yaml:"prompt"`
	Tools       []string `yaml:"tools,omitempty"`
}

func LoadCustomCommands() ([]CustomCommand, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	path := filepath.Join(home, ".supcode", "commands.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read commands: %w", err)
	}

	var cfg CustomCommandsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse commands: %w", err)
	}
	return cfg.Commands, nil
}

func SaveCustomCommands(commands []CustomCommand) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".supcode")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	cfg := CustomCommandsConfig{Commands: commands}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	path := filepath.Join(dir, "commands.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}
