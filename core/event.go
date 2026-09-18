package core

import (
	"context"
	"errors"
	"sync"
)

// Handler is an event listener. The payload and return value are type-erased;
// typed dispatch helpers (Waterfall/Serial/Bail) perform the conversion.
//
// next continues the handler chain and must be called (zero or more times) by
// waterfall handlers that want to wrap the remaining chain. Emit/parallel
// dispatch passes a no-op next.
type Handler func(ctx context.Context, payload any, next Next) (any, error)

// Next continues a waterfall handler chain with a possibly modified payload.
type Next func(payload any) (any, error)

// EventOption customizes a listener registration.
type EventOption func(*listener)

// Once removes the listener after its first invocation.
func Once() EventOption {
	return func(l *listener) { l.once = true }
}

type listener struct {
	id    uint64
	event string
	fn    Handler
	once  bool
}

// addListener appends h to the event's listener list and returns a removal
// function. Listener order is registration order.
func (k *Kernel) addListener(event string, h Handler, opts ...EventOption) func() {
	l := &listener{fn: h, event: event}
	for _, opt := range opts {
		opt(l)
	}
	k.mu.Lock()
	l.id = k.nextListenerID
	k.nextListenerID++
	k.listeners[event] = append(k.listeners[event], l)
	k.mu.Unlock()

	return func() { k.removeListener(l) }
}

func (k *Kernel) removeListener(l *listener) {
	k.mu.Lock()
	defer k.mu.Unlock()
	ls := k.listeners[l.event]
	for i, x := range ls {
		if x == l {
			k.listeners[l.event] = append(ls[:i], ls[i+1:]...)
			break
		}
	}
}

// handlers returns a snapshot of the listeners registered for event.
func (k *Kernel) handlers(event string) []*listener {
	k.mu.RLock()
	defer k.mu.RUnlock()
	ls := k.listeners[event]
	out := make([]*listener, len(ls))
	copy(out, ls)
	return out
}

func noopNext(payload any) (any, error) { return payload, nil }

// Emit notifies listeners in registration order and ignores their results.
// It is fire-and-forget (no awaiting, no return value).
func Emit(ctx context.Context, k *Kernel, event string, payload any) {
	for _, l := range k.handlers(event) {
		_, _ = l.fn(ctx, payload, noopNext)
		if l.once {
			k.removeListener(l)
		}
	}
}

// Waterfall dispatches event as an around-middleware chain. Each handler may
// short-circuit by not calling next, or wrap the remainder by calling next and
// post-processing the result. The payload type Req and result type Resp are
// converted automatically.
func Waterfall[Req, Resp any](ctx context.Context, k *Kernel, event string, req Req) (Resp, error) {
	var zero Resp
	ls := k.handlers(event)

	var call func(i int, payload any) (any, error)
	call = func(i int, payload any) (any, error) {
		if i >= len(ls) {
			return payload, nil
		}
		l := ls[i]
		next := func(np any) (any, error) { return call(i+1, np) }
		res, err := l.fn(ctx, payload, next)
		if l.once {
			k.removeListener(l)
		}
		return res, err
	}

	res, err := call(0, any(req))
	if err != nil {
		return zero, err
	}
	if res == nil {
		return zero, nil
	}
	typed, ok := res.(Resp)
	if !ok {
		return zero, errors.New("core: waterfall result type mismatch")
	}
	return typed, nil
}

// Parallel fans out to all listeners concurrently and waits for completion.
// It returns the joined error of every failing listener.
func Parallel(ctx context.Context, k *Kernel, event string, payload any) error {
	ls := k.handlers(event)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, l := range ls {
		l := l
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := l.fn(ctx, payload, noopNext); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
			if l.once {
				k.removeListener(l)
			}
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

// Serial invokes listeners sequentially. A non-nil result becomes the payload
// of the next listener; the last non-nil result is returned.
func Serial[Resp any](ctx context.Context, k *Kernel, event string, payload any) (Resp, error) {
	var zero Resp
	cur := payload
	for _, l := range k.handlers(event) {
		res, err := l.fn(ctx, cur, noopNext)
		if l.once {
			k.removeListener(l)
		}
		if err != nil {
			return zero, err
		}
		if res != nil {
			cur = res
			if typed, ok := res.(Resp); ok {
				zero = typed
			}
		}
	}
	return zero, nil
}

// Bail invokes listeners in order and stops at the first listener that returns
// a non-nil result. It returns that result with bail=true, or the zero value
// with bail=false when no listener decided.
func Bail[Resp any](ctx context.Context, k *Kernel, event string, payload any) (Resp, bool, error) {
	var zero Resp
	for _, l := range k.handlers(event) {
		res, err := l.fn(ctx, payload, noopNext)
		if l.once {
			k.removeListener(l)
		}
		if err != nil {
			return zero, false, err
		}
		if res != nil {
			if typed, ok := res.(Resp); ok {
				zero = typed
			}
			return zero, true, nil
		}
	}
	return zero, false, nil
}
