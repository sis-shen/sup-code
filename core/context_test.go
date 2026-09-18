package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvideUseAndDisposer(t *testing.T) {
	k := New()
	c := k.Root()

	d := Provide(c, "answer", 42)
	assert.Equal(t, 42, Use[int](c, "answer"))
	assert.True(t, c.Has("answer"))

	require.NoError(t, d())
	_, ok := MaybeUse[int](c, "answer")
	assert.False(t, ok)
	assert.False(t, c.Has("answer"))
}

func TestProvideRestoresPrevious(t *testing.T) {
	c := New().Root()
	Provide(c, "x", 1)

	d := Provide(c, "x", 2)
	assert.Equal(t, 2, Use[int](c, "x"))

	require.NoError(t, d())
	assert.Equal(t, 1, Use[int](c, "x"))
}

func TestProvidePanicsOnEmptyKey(t *testing.T) {
	assert.Panics(t, func() { New().Root().Provide("", 1) })
}

func TestUsePanicsOnMissing(t *testing.T) {
	assert.Panics(t, func() { Use[int](New().Root(), "nope") })
}

func TestUsePanicsOnWrongType(t *testing.T) {
	c := New().Root()
	Provide(c, "x", 1)
	assert.Panics(t, func() { Use[string](c, "x") })
}

func TestMaybeUseWrongType(t *testing.T) {
	c := New().Root()
	Provide(c, "x", 1)
	_, ok := MaybeUse[string](c, "x")
	assert.False(t, ok)
}

func TestForkInheritsParent(t *testing.T) {
	k := New()
	c := k.Root()
	Provide(c, "svc", "hello")

	child := c.Fork("child")
	assert.Equal(t, "hello", Use[string](child, "svc"))
	assert.Equal(t, "child", child.Name())
	assert.Same(t, c, child.Parent())
	assert.Same(t, k, child.Kernel())
	assert.NotSame(t, c, child)
}

func TestIsolateHidesParent(t *testing.T) {
	k := New()
	c := k.Root()
	Provide(c, "svc", "hello")

	iso := c.Isolate("iso")
	assert.False(t, iso.Has("svc"))
	assert.Panics(t, func() { Use[string](iso, "svc") })

	Provide(iso, "svc", "inner")
	assert.Equal(t, "inner", Use[string](iso, "svc"))
}

func TestEffectReverseOrder(t *testing.T) {
	c := New().Root()
	var order []string

	require.NoError(t, c.Effect(func() (Disposer, error) {
		return func() error { order = append(order, "first"); return nil }, nil
	}))
	require.NoError(t, c.Effect(func() (Disposer, error) {
		return func() error { order = append(order, "second"); return nil }, nil
	}))

	require.NoError(t, c.Dispose())
	assert.Equal(t, []string{"second", "first"}, order)
}

func TestEffectNilFunction(t *testing.T) {
	assert.Error(t, New().Root().Effect(nil))
}

func TestEffectSetupError(t *testing.T) {
	sentinel := errors.New("boom")
	err := New().Root().Effect(func() (Disposer, error) { return nil, sentinel })
	assert.ErrorIs(t, err, sentinel)
}

func TestEffectNilDisposer(t *testing.T) {
	c := New().Root()
	require.NoError(t, c.Effect(func() (Disposer, error) { return nil, nil }))
	require.NoError(t, c.Dispose())
}

func TestEffectAfterDispose(t *testing.T) {
	c := New().Root()
	require.NoError(t, c.Dispose())
	assert.Error(t, c.Effect(func() (Disposer, error) { return nil, nil }))
}

func TestDisposeAggregatesErrorsAndIsIdempotent(t *testing.T) {
	c := New().Root()
	boom := errors.New("boom")
	require.NoError(t, c.Effect(func() (Disposer, error) {
		return func() error { return boom }, nil
	}))

	err := c.Dispose()
	assert.ErrorIs(t, err, boom)
	assert.True(t, c.Disposed())
	assert.NoError(t, c.Dispose())
}

func TestOnRemovedOnDispose(t *testing.T) {
	k := New()
	c := k.Root()
	called := 0
	c.On("evt", func(_ context.Context, _ any, _ Next) (any, error) {
		called++
		return nil, nil
	})

	Emit(context.Background(), k, "evt", nil)
	assert.Equal(t, 1, called)

	require.NoError(t, c.Dispose())
	Emit(context.Background(), k, "evt", nil)
	assert.Equal(t, 1, called)
}

func TestOnPanicsOnNilHandler(t *testing.T) {
	assert.Panics(t, func() { New().Root().On("evt", nil) })
}

func TestOnAfterDisposeIsIgnored(t *testing.T) {
	k := New()
	c := k.Root()
	require.NoError(t, c.Dispose())

	called := 0
	d := c.On("evt", func(_ context.Context, _ any, _ Next) (any, error) {
		called++
		return nil, nil
	})
	require.NoError(t, d())
	Emit(context.Background(), k, "evt", nil)
	assert.Equal(t, 0, called)
}

func TestContextAccessors(t *testing.T) {
	k := New()
	c := k.Root()
	assert.Same(t, k, c.Kernel())
	assert.Equal(t, "root", c.Name())
	assert.Nil(t, c.Parent())
	assert.Nil(t, c.Config())
	assert.False(t, c.Disposed())
}
