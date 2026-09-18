// Package core implements the Cordis-semantic micro-kernel for Sup Harness 2.0.
//
// The kernel is the only non-plugin component: it owns the service registry
// (Context), the typed event bus (emit/waterfall/parallel/serial/bail),
// scoped lifecycle with reversible effects (Fork/Isolate/Effect), and the
// plugin loader (inject topology, Load/Unload/Reload). It must never import
// plugin/* or internal/*; all capabilities reach the kernel through services
// and events.
package core
