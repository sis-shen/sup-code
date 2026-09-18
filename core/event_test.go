package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmitOrderAndOnce(t *testing.T) {
	k := New()
	c := k.Root()
	var order []string

	c.On("e", func(_ context.Context, _ any, _ Next) (any, error) {
		order = append(order, "a")
		return nil, nil
	})
	c.On("e", func(_ context.Context, _ any, _ Next) (any, error) {
		order = append(order, "b")
		return nil, nil
	}, Once())

	Emit(context.Background(), k, "e", nil)
	Emit(context.Background(), k, "e", nil)

	assert.Equal(t, []string{"a", "b", "a"}, order)
}

func TestOnDisposerRemovesListener(t *testing.T) {
	k := New()
	called := 0
	d := k.Root().On("e", func(_ context.Context, _ any, _ Next) (any, error) {
		called++
		return nil, nil
	})

	require.NoError(t, d())
	Emit(context.Background(), k, "e", nil)
	assert.Equal(t, 0, called)
}

func TestWaterfallWraps(t *testing.T) {
	k := New()
	k.Root().On("calc", func(_ context.Context, p any, next Next) (any, error) {
		res, err := next(p.(int) + 1)
		if err != nil {
			return nil, err
		}
		return res.(int) * 2, nil
	})
	k.Root().On("calc", func(_ context.Context, p any, _ Next) (any, error) {
		return p.(int) + 10, nil
	})

	got, err := Waterfall[int, int](context.Background(), k, "calc", 1)
	require.NoError(t, err)
	assert.Equal(t, 24, got) // ((1+1)+10)*2
}

func TestWaterfallShortCircuit(t *testing.T) {
	k := New()
	k.Root().On("calc", func(_ context.Context, _ any, _ Next) (any, error) {
		return 99, nil
	})
	k.Root().On("calc", func(_ context.Context, _ any, _ Next) (any, error) {
		return 1, nil
	})

	got, err := Waterfall[int, int](context.Background(), k, "calc", 1)
	require.NoError(t, err)
	assert.Equal(t, 99, got)
}

func TestWaterfallNoListeners(t *testing.T) {
	got, err := Waterfall[int, int](context.Background(), New(), "none", 5)
	require.NoError(t, err)
	assert.Equal(t, 5, got)
}

func TestWaterfallError(t *testing.T) {
	k := New()
	sentinel := errors.New("x")
	k.Root().On("e", func(_ context.Context, _ any, _ Next) (any, error) {
		return nil, sentinel
	})

	_, err := Waterfall[int, int](context.Background(), k, "e", 1)
	assert.ErrorIs(t, err, sentinel)
}

func TestWaterfallTypeMismatch(t *testing.T) {
	k := New()
	k.Root().On("e", func(_ context.Context, _ any, _ Next) (any, error) {
		return "string", nil
	})

	_, err := Waterfall[int, int](context.Background(), k, "e", 1)
	assert.Error(t, err)
}

func TestParallelCollectsErrors(t *testing.T) {
	k := New()
	e1 := errors.New("e1")
	e2 := errors.New("e2")
	h := func(err error) Handler {
		return func(_ context.Context, _ any, _ Next) (any, error) { return nil, err }
	}
	k.Root().On("p", h(e1))
	k.Root().On("p", h(nil))
	k.Root().On("p", h(e2))

	err := Parallel(context.Background(), k, "p", nil)
	assert.ErrorIs(t, err, e1)
	assert.ErrorIs(t, err, e2)
}

func TestParallelNoListeners(t *testing.T) {
	assert.NoError(t, Parallel(context.Background(), New(), "p", nil))
}

func TestSerialFeedsPayloadAndReturnsLast(t *testing.T) {
	k := New()
	k.Root().On("s", func(_ context.Context, p any, _ Next) (any, error) {
		return p.(int) + 1, nil
	})
	k.Root().On("s", func(_ context.Context, p any, _ Next) (any, error) {
		return p.(int) * 10, nil
	})

	got, err := Serial[int](context.Background(), k, "s", 1)
	require.NoError(t, err)
	assert.Equal(t, 20, got)
}

func TestSerialError(t *testing.T) {
	k := New()
	sentinel := errors.New("x")
	k.Root().On("s", func(_ context.Context, _ any, _ Next) (any, error) {
		return nil, sentinel
	})

	_, err := Serial[int](context.Background(), k, "s", 1)
	assert.ErrorIs(t, err, sentinel)
}

func TestBailReturnsFirstDecision(t *testing.T) {
	k := New()
	k.Root().On("pick", func(_ context.Context, _ any, _ Next) (any, error) { return nil, nil })
	k.Root().On("pick", func(_ context.Context, _ any, _ Next) (any, error) { return "chosen", nil })
	k.Root().On("pick", func(_ context.Context, _ any, _ Next) (any, error) { return "later", nil })

	got, bail, err := Bail[string](context.Background(), k, "pick", nil)
	require.NoError(t, err)
	assert.True(t, bail)
	assert.Equal(t, "chosen", got)
}

func TestBailNoDecision(t *testing.T) {
	k := New()
	k.Root().On("pick", func(_ context.Context, _ any, _ Next) (any, error) { return nil, nil })

	got, bail, err := Bail[string](context.Background(), k, "pick", nil)
	require.NoError(t, err)
	assert.False(t, bail)
	assert.Empty(t, got)
}

func TestBailError(t *testing.T) {
	k := New()
	sentinel := errors.New("x")
	k.Root().On("pick", func(_ context.Context, _ any, _ Next) (any, error) {
		return nil, sentinel
	})

	_, bail, err := Bail[string](context.Background(), k, "pick", nil)
	assert.False(t, bail)
	assert.ErrorIs(t, err, sentinel)
}
