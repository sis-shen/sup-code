package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

// mockHook implements pkg.ToolHook for testing.
type mockHook struct {
	name           string
	beforeErr      error
	afterErr       error
	beforeDelay    time.Duration
	afterDelay     time.Duration
	beforeCalled   *int
	afterCalled    *int
	modifyParams   bool
}

func newMockHook(name string) *mockHook {
	return &mockHook{
		name:         name,
		beforeCalled: new(int),
		afterCalled:  new(int),
	}
}

func (h *mockHook) Name() string { return h.name }

func (h *mockHook) BeforeTool(ctx context.Context, toolName string, params json.RawMessage) (json.RawMessage, error) {
	*h.beforeCalled++
	if h.beforeDelay > 0 {
		select {
		case <-time.After(h.beforeDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if h.beforeErr != nil {
		return nil, h.beforeErr
	}
	if h.modifyParams {
		return json.RawMessage(`{"modified":true}`), nil
	}
	return params, nil
}

func (h *mockHook) AfterTool(ctx context.Context, toolName string, params json.RawMessage, result pkg.ToolResult) error {
	*h.afterCalled++
	if h.afterDelay > 0 {
		select {
		case <-time.After(h.afterDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return h.afterErr
}

func TestHookEngine_Register_List(t *testing.T) {
	eng := NewHookEngine()
	assert.Empty(t, eng.List())

	h1 := newMockHook("hook1")
	h2 := newMockHook("hook2")

	require.NoError(t, eng.Register(h1))
	require.NoError(t, eng.Register(h2))
	assert.ElementsMatch(t, []string{"hook1", "hook2"}, eng.List())

	// Duplicate
	err := eng.Register(h1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestHookEngine_Register_Nil(t *testing.T) {
	eng := NewHookEngine()
	err := eng.Register(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil hook")
}

func TestHookEngine_Register_EmptyName(t *testing.T) {
	eng := NewHookEngine()
	err := eng.Register(newMockHook(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestHookEngine_Unregister(t *testing.T) {
	eng := NewHookEngine()
	require.NoError(t, eng.Register(newMockHook("hook1")))
	require.NoError(t, eng.Register(newMockHook("hook2")))

	require.NoError(t, eng.Unregister("hook1"))
	assert.ElementsMatch(t, []string{"hook2"}, eng.List())

	// Already removed
	err := eng.Unregister("hook1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestHookEngine_ExecuteBefore_AllPass(t *testing.T) {
	eng := NewHookEngine()
	h1 := newMockHook("h1")
	h2 := newMockHook("h2")
	require.NoError(t, eng.Register(h1))
	require.NoError(t, eng.Register(h2))

	params := json.RawMessage(`{"key":"value"}`)
	result, err := eng.ExecuteBefore(context.Background(), "test_tool", params)
	require.NoError(t, err)
	assert.Equal(t, params, result)
	assert.Equal(t, 1, *h1.beforeCalled)
	assert.Equal(t, 1, *h2.beforeCalled)
}

func TestHookEngine_ExecuteBefore_FirstRejects(t *testing.T) {
	eng := NewHookEngine()
	h1 := newMockHook("h1")
	h1.beforeErr = errors.New("rejected")
	h2 := newMockHook("h2")
	require.NoError(t, eng.Register(h1))
	require.NoError(t, eng.Register(h2))

	_, err := eng.ExecuteBefore(context.Background(), "test_tool", json.RawMessage(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rejected")
	// h2 should NOT be called
	assert.Equal(t, 0, *h2.beforeCalled)
}

func TestHookEngine_ExecuteBefore_ModifiesParams(t *testing.T) {
	eng := NewHookEngine()
	h1 := newMockHook("h1")
	h1.modifyParams = true
	h2 := newMockHook("h2")
	require.NoError(t, eng.Register(h1))
	require.NoError(t, eng.Register(h2))

	result, err := eng.ExecuteBefore(context.Background(), "test_tool", json.RawMessage(`{"original":true}`))
	require.NoError(t, err)
	assert.Contains(t, string(result), `"modified"`)
}

func TestHookEngine_ExecuteBefore_ContextCancellation(t *testing.T) {
	eng := NewHookEngine()
	h1 := newMockHook("h1")
	h1.beforeDelay = 5 * time.Second
	require.NoError(t, eng.Register(h1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	time.Sleep(20 * time.Millisecond)

	_, err := eng.ExecuteBefore(ctx, "test_tool", json.RawMessage(`{}`))
	require.Error(t, err)
}

func TestHookEngine_ExecuteAfter_AllPass(t *testing.T) {
	eng := NewHookEngine()
	h1 := newMockHook("h1")
	h2 := newMockHook("h2")
	require.NoError(t, eng.Register(h1))
	require.NoError(t, eng.Register(h2))

	result := pkg.ToolResult{Success: true, Data: json.RawMessage(`{"ok":true}`)}
	eng.ExecuteAfter(context.Background(), "test_tool", json.RawMessage(`{}`), result)
	assert.Equal(t, 1, *h1.afterCalled)
	assert.Equal(t, 1, *h2.afterCalled)
}

func TestHookEngine_ExecuteAfter_ErrorDoesNotPanic(t *testing.T) {
	eng := NewHookEngine()
	h1 := newMockHook("h1")
	h1.afterErr = errors.New("after error")
	h2 := newMockHook("h2")
	require.NoError(t, eng.Register(h1))
	require.NoError(t, eng.Register(h2))

	result := pkg.ToolResult{Success: true}
	// Should not panic even if h1 returns error
	eng.ExecuteAfter(context.Background(), "test_tool", json.RawMessage(`{}`), result)
	assert.Equal(t, 1, *h1.afterCalled)
	assert.Equal(t, 1, *h2.afterCalled, "h2 should still be called after h1 error")
}

func TestHookEngine_ConcurrentSafe(t *testing.T) {
	eng := NewHookEngine()
	var wg sync.WaitGroup

	// Concurrently register hooks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			hook := newMockHook(fmt.Sprintf("hook_%d", n))
			_ = eng.Register(hook)
		}(i)
	}

	wg.Wait()

	// Concurrently execute before/after
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = eng.ExecuteBefore(context.Background(), "test", json.RawMessage(`{}`))
			eng.ExecuteAfter(context.Background(), "test", json.RawMessage(`{}`), pkg.ToolResult{Success: true})
		}()
	}
	wg.Wait()

	names := eng.List()
	assert.GreaterOrEqual(t, len(names), 1)
}

func TestHookEngine_HookWrapper(t *testing.T) {
	h := newMockHook("test_hook")
	w := NewHookWrapper(h)
	assert.Equal(t, "test_hook", w.Name())

	// Disable
	w.Disable()
	params := json.RawMessage(`{"k":"v"}`)
	result, err := w.BeforeTool(context.Background(), "tool", params)
	require.NoError(t, err)
	assert.Equal(t, params, result)
	assert.Equal(t, 0, *h.beforeCalled) // not called when disabled

	err = w.AfterTool(context.Background(), "tool", params, pkg.ToolResult{Success: true})
	require.NoError(t, err)
	assert.Equal(t, 0, *h.afterCalled)

	// Enable
	w.Enable()
	_, _ = w.BeforeTool(context.Background(), "tool", params)
	assert.Equal(t, 1, *h.beforeCalled)
}

func TestHookEngine_ExecuteBefore_SecondHookFails(t *testing.T) {
	eng := NewHookEngine()
	h1 := newMockHook("h1")
	h1.modifyParams = true
	h2 := newMockHook("h2")
	h2.beforeErr = errors.New("h2 rejected")
	h3 := newMockHook("h3")
	require.NoError(t, eng.Register(h1))
	require.NoError(t, eng.Register(h2))
	require.NoError(t, eng.Register(h3))

	_, err := eng.ExecuteBefore(context.Background(), "test_tool", json.RawMessage(`{"original":true}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "h2 rejected")

	// h1 ran (modified params), h2 failed, h3 should NOT run
	assert.Equal(t, 1, *h1.beforeCalled)
	assert.Equal(t, 1, *h2.beforeCalled)
	assert.Equal(t, 0, *h3.beforeCalled)
}

// Prevent unused import
var _ = pkg.ToolHook(nil)
