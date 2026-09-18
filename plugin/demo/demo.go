// Package demo provides a minimal plugin set that exercises every kernel
// primitive (Provide/Inject, events, effects and scoped lifecycle). It doubles
// as a template for real plugins.
package demo

import (
	"context"
	"fmt"
	"sync"

	"github.com/supcode/supcode/core"
)

// Logger is the service provided by the demo-logger plugin.
type Logger struct {
	mu    sync.Mutex
	lines []string
}

// NewLogger creates an empty logger.
func NewLogger() *Logger { return &Logger{} }

// Log appends a line.
func (l *Logger) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, msg)
}

// Lines returns a copy of the recorded lines.
func (l *Logger) Lines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.lines...)
}

// Counter is the service provided by the demo-counter plugin.
type Counter struct {
	logger *Logger
	mu     sync.Mutex
	n      int
}

// Add increments the counter and returns the new value.
func (c *Counter) Add(delta int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n += delta
	return c.n
}

// Value returns the current value.
func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// LoggerPlugin provides the "logger" service.
func LoggerPlugin() core.Plugin {
	return core.Plugin{
		Name:     "demo-logger",
		Provides: []string{"logger"},
		Apply: func(ctx *core.Context) error {
			logger := NewLogger()
			core.Provide(ctx, "logger", logger)
			return ctx.Effect(func() (core.Disposer, error) {
				return func() error {
					logger.Log("logger disposed")
					return nil
				}, nil
			})
		},
	}
}

// CounterPlugin injects "logger", provides "counter" and listens for the
// "demo/add" event.
func CounterPlugin() core.Plugin {
	return core.Plugin{
		Name:     "demo-counter",
		Inject:   []string{"logger"},
		Provides: []string{"counter"},
		Config:   func() any { return map[string]any{"start": 0} },
		Apply: func(ctx *core.Context) error {
			logger := core.Use[*Logger](ctx, "logger")
			counter := &Counter{logger: logger}
			core.Provide(ctx, "counter", counter)

			ctx.On("demo/add", func(_ context.Context, payload any, _ core.Next) (any, error) {
				delta, ok := payload.(int)
				if !ok {
					return nil, fmt.Errorf("demo/add: expected int payload, got %T", payload)
				}
				total := counter.Add(delta)
				logger.Log(fmt.Sprintf("add %d -> %d", delta, total))
				return total, nil
			})

			return ctx.Effect(func() (core.Disposer, error) {
				return func() error {
					logger.Log("counter disposed")
					return nil
				}, nil
			})
		},
	}
}

// Plugins returns the demo plugin set (in an order that LoadPlugins resolves
// to a valid dependency order).
func Plugins() []core.Plugin {
	return []core.Plugin{CounterPlugin(), LoggerPlugin()}
}
