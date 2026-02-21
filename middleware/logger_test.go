package middleware

import (
	"fmt"
	"strings"
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
	maxigobot "github.com/maxigo-bot/maxigo-bot"
)

func TestLogger(t *testing.T) {
	var logged string
	mw := LoggerWithConfig(LoggerConfig{
		Log: func(format string, args ...any) {
			logged = fmt.Sprintf(format, args...)
		},
	})

	ctx := &mockContext{
		update: maxigo.Update{UpdateType: maxigo.UpdateMessageCreated},
		sender: &maxigo.User{UserID: 42},
		chatID: 100,
	}

	handler := mw(func(c maxigobot.Context) error {
		return nil
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logged == "" {
		t.Fatal("nothing was logged")
	}
	if !strings.Contains(logged, "message_created") {
		t.Errorf("log should contain update type, got: %s", logged)
	}
	if !strings.Contains(logged, "42") {
		t.Errorf("log should contain sender ID, got: %s", logged)
	}
	if !strings.Contains(logged, "100") {
		t.Errorf("log should contain chat ID, got: %s", logged)
	}
}

func TestLogger_withError(t *testing.T) {
	var logged string
	mw := LoggerWithConfig(LoggerConfig{
		Log: func(format string, args ...any) {
			logged = fmt.Sprintf(format, args...)
		},
	})

	ctx := &mockContext{
		update: maxigo.Update{UpdateType: maxigo.UpdateBotStarted},
		sender: &maxigo.User{UserID: 1},
		chatID: 10,
	}

	handler := mw(func(c maxigobot.Context) error {
		return errForTest("something failed")
	})

	err := handler(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(logged, "error: something failed") {
		t.Errorf("log should contain error, got: %s", logged)
	}
}

func TestLogger_nilSender(t *testing.T) {
	var logged string
	mw := LoggerWithConfig(LoggerConfig{
		Log: func(format string, args ...any) {
			logged = fmt.Sprintf(format, args...)
		},
	})

	ctx := &mockContext{
		update: maxigo.Update{UpdateType: maxigo.UpdateMessageRemoved},
	}

	handler := mw(func(c maxigobot.Context) error {
		return nil
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// sender=0 when nil.
	if !strings.Contains(logged, "sender=0") {
		t.Errorf("log should contain sender=0 for nil sender, got: %s", logged)
	}
}

func TestLogger_skipper(t *testing.T) {
	logCalled := false
	mw := LoggerWithConfig(LoggerConfig{
		Skipper: func(_ maxigobot.Context) bool { return true },
		Log: func(format string, args ...any) {
			logCalled = true
		},
	})

	handlerCalled := false
	handler := mw(func(c maxigobot.Context) error {
		handlerCalled = true
		return nil
	})

	err := handler(&mockContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler should still be called when skipped")
	}
	if logCalled {
		t.Error("log should not be called when skipped")
	}
}

func TestLogger_default(t *testing.T) {
	// Logger() with defaults should not panic.
	mw := Logger()
	handler := mw(func(c maxigobot.Context) error {
		return nil
	})

	err := handler(&mockContext{
		update: maxigo.Update{UpdateType: maxigo.UpdateBotStarted},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
