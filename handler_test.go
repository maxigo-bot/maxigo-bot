package maxigobot

import (
	"testing"
)

func TestApplyMiddleware_empty(t *testing.T) {
	called := false
	h := func(c Context) error {
		called = true
		return nil
	}

	result := applyMiddleware(h)
	if err := result(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestApplyMiddleware_order(t *testing.T) {
	var order []int

	mw := func(id int) MiddlewareFunc {
		return func(next HandlerFunc) HandlerFunc {
			return func(c Context) error {
				order = append(order, id)
				return next(c)
			}
		}
	}

	h := func(c Context) error {
		order = append(order, 0)
		return nil
	}

	result := applyMiddleware(h, mw(1), mw(2), mw(3))
	if err := result(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Middleware 1 should execute first (outermost), then 2, then 3, then handler (0).
	expected := []int{1, 2, 3, 0}
	if len(order) != len(expected) {
		t.Fatalf("execution order length = %d, want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %d, want %d", i, order[i], v)
		}
	}
}

func TestApplyMiddleware_shortCircuit(t *testing.T) {
	handlerCalled := false

	blocker := func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			// Don't call next — short-circuit.
			return nil
		}
	}

	h := func(c Context) error {
		handlerCalled = true
		return nil
	}

	result := applyMiddleware(h, blocker)
	if err := result(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handlerCalled {
		t.Fatal("handler should not have been called")
	}
}
