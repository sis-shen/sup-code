// Package llm exposes the v1 LLM provider clients (openai / anthropic /
// deepseek / ollama) as a Cordis leaf plugin.
//
// The plugin adapts the existing internal/llm factory without reimplementing
// any provider logic: Apply constructs a pkg.LLMClient (or accepts one
// directly) and registers it under pkg.ServiceLLM. Configuration is optional
// and only read when present on the kernel, so the plugin stays loadable
// before plugin-config exists.
package llm

import (
	"github.com/supcode/supcode/core"
	llmfactory "github.com/supcode/supcode/internal/llm"
	"github.com/supcode/supcode/pkg"
)

const (
	pluginName    = "plugin-llm"
	pluginVersion = "2.0.0"
)

// Config keys mirror internal/config (llm.*). They are duplicated as literals
// so this plugin does not depend on a concrete config provider.
const (
	configKeyProvider  = "llm.provider"
	configKeyAPIKey    = "llm.api_key"
	configKeyBaseURL   = "llm.base_url"
	configKeyModel     = "llm.model"
	configKeyMaxTokens = "llm.max_tokens"
)

// Options configures the LLM plugin. A non-nil Client takes precedence and is
// registered as-is; otherwise a client is built from the provider fields via
// the v1 factory.
type Options struct {
	Client    pkg.LLMClient
	Provider  string
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
}

// Plugin returns the loadable LLM plugin. It provides pkg.ServiceLLM and has
// no hard dependencies; the config service is read opportunistically.
func Plugin(opts Options) core.Plugin {
	return core.Plugin{
		Name:     pluginName,
		Provides: []string{pkg.ServiceLLM},
		Apply: func(ctx *core.Context) error {
			resolved := resolveOptions(opts, ctx)

			client := resolved.Client
			if client == nil {
				built, err := llmfactory.NewLLMClient(llmfactory.Config{
					Provider:  resolved.Provider,
					APIKey:    resolved.APIKey,
					BaseURL:   resolved.BaseURL,
					Model:     resolved.Model,
					MaxTokens: resolved.MaxTokens,
				})
				if err != nil {
					return err
				}
				client = built
			}

			core.Provide(ctx, pkg.ServiceLLM, client)
			return nil
		},
	}
}

// resolveOptions fills empty option fields from the optional config service.
// Fields explicitly set on opts always win.
func resolveOptions(opts Options, ctx *core.Context) Options {
	cfg, ok := core.MaybeUse[pkg.Config](ctx, pkg.ServiceConfig)
	if !ok || cfg == nil {
		return opts
	}
	if opts.Provider == "" {
		opts.Provider = cfg.GetString(configKeyProvider)
	}
	if opts.APIKey == "" {
		opts.APIKey = cfg.GetString(configKeyAPIKey)
	}
	if opts.BaseURL == "" {
		opts.BaseURL = cfg.GetString(configKeyBaseURL)
	}
	if opts.Model == "" {
		opts.Model = cfg.GetString(configKeyModel)
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = cfg.GetInt(configKeyMaxTokens)
	}
	return opts
}

// Manifest describes the plugin and mirrors sup.plugin.json.
func Manifest() core.Manifest {
	return core.Manifest{
		Name:        pluginName,
		Version:     pluginVersion,
		Description: "LLM provider clients exposed as the llm service",
		Provides:    []string{pkg.ServiceLLM},
		Entry:       "plugin/llm",
	}
}

// AvailableProviders returns the provider names supported by the v1 factory.
func AvailableProviders() []string {
	return llmfactory.AvailableProviders()
}
