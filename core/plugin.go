package core

// Plugin is a declarative unit of functionality. Everything outside the kernel
// is a plugin.
//
// A plugin declares the service keys it consumes via Inject, optionally
// provides a default configuration value, and implements Apply to install its
// services, event listeners and effects onto the supplied scope. Applying to a
// child scope means every registration is reversible: unloading the plugin
// disposes the scope and replays its effects in reverse order.
type Plugin struct {
	// Name uniquely identifies the plugin within a kernel.
	Name string
	// Inject lists service keys that must be available before Apply runs.
	Inject []string
	// Provides lists the service keys this plugin registers. It is used by the
	// loader to order plugins so that providers start before consumers.
	Provides []string
	// Config returns the plugin's default configuration, or nil.
	Config func() any
	// Apply installs the plugin onto its scope.
	Apply func(ctx *Context) error
}

// Manifest is the serializable description of a plugin (sup.plugin.json). It
// mirrors the Cordis plugin declaration so that external manifests can be
// mapped onto Plugin values.
type Manifest struct {
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Inject      []string `json:"inject,omitempty"`
	Provides    []string `json:"provides,omitempty"`
	Entry       string   `json:"entry,omitempty"`
}
