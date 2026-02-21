package middleware

import (
	"strings"
	"testing"

	maxigobot "github.com/maxigo-bot/maxigo-bot"
)

func TestRecover(t *testing.T) {
	mw := Recover()
	handler := mw(func(c maxigobot.Context) error {
		panic("test panic")
	})

	err := handler(&mockContext{})
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}
	if !strings.Contains(err.Error(), "panic recovered: test panic") {
		t.Errorf("unexpected error: %v", err)
	}
	// Default config includes stack trace.
	if !strings.Contains(err.Error(), "goroutine") {
		t.Error("expected stack trace in error")
	}
}

func TestRecover_noPanic(t *testing.T) {
	called := false
	mw := Recover()
	handler := mw(func(c maxigobot.Context) error {
		called = true
		return nil
	})

	err := handler(&mockContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestRecover_propagatesError(t *testing.T) {
	mw := Recover()
	want := "handler error"
	handler := mw(func(c maxigobot.Context) error {
		return errForTest(want)
	})

	err := handler(&mockContext{})
	if err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}
}

func TestRecoverWithConfig_noStack(t *testing.T) {
	mw := RecoverWithConfig(RecoverConfig{
		PrintStack: false,
	})
	handler := mw(func(c maxigobot.Context) error {
		panic("no stack")
	})

	err := handler(&mockContext{})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "goroutine") {
		t.Error("stack trace should not be included")
	}
	if !strings.Contains(err.Error(), "panic recovered: no stack") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRecoverWithConfig_skipper(t *testing.T) {
	mw := RecoverWithConfig(RecoverConfig{
		Skipper: func(_ maxigobot.Context) bool { return true },
	})

	// When skipped, panic is not recovered by this middleware.
	handler := mw(func(c maxigobot.Context) error {
		return nil
	})

	err := handler(&mockContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecoverWithConfig_stackSizeLimit(t *testing.T) {
	mw := RecoverWithConfig(RecoverConfig{
		StackSize:  100, // very small
		PrintStack: true,
	})
	handler := mw(func(c maxigobot.Context) error {
		panic("limited stack")
	})

	err := handler(&mockContext{})
	if err == nil {
		t.Fatal("expected error")
	}
	// The stack portion should be truncated, but the prefix is still there.
	if !strings.Contains(err.Error(), "panic recovered: limited stack") {
		t.Errorf("unexpected error: %v", err)
	}
}

// errForTest is a simple error for testing.
type errForTest string

func (e errForTest) Error() string { return string(e) }
