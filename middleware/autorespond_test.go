package middleware

import (
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
	maxigobot "github.com/maxigo-bot/maxigo-bot"
)

func TestAutoRespond_callback(t *testing.T) {
	mw := AutoRespond()
	ctx := &mockContext{
		callback: &maxigo.Callback{CallbackID: "cb1", Payload: "confirm"},
	}

	handlerCalled := false
	handler := mw(func(c maxigobot.Context) error {
		handlerCalled = true
		return nil
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler was not called")
	}
	if !ctx.respondCalled {
		t.Error("Respond should be called for callback updates")
	}
	if ctx.respondText != "" {
		t.Errorf("respondText = %q, want empty", ctx.respondText)
	}
}

func TestAutoRespond_nonCallback(t *testing.T) {
	mw := AutoRespond()
	ctx := &mockContext{}

	handler := mw(func(c maxigobot.Context) error {
		return nil
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.respondCalled {
		t.Error("Respond should not be called for non-callback updates")
	}
}

func TestAutoRespond_handlerError(t *testing.T) {
	mw := AutoRespond()
	ctx := &mockContext{
		callback: &maxigo.Callback{CallbackID: "cb2"},
	}

	handler := mw(func(c maxigobot.Context) error {
		return errForTest("handler failed")
	})

	err := handler(ctx)
	if err == nil || err.Error() != "handler failed" {
		t.Errorf("error = %v, want 'handler failed'", err)
	}
	// AutoRespond should still call Respond even when handler returns error.
	if !ctx.respondCalled {
		t.Error("Respond should be called even when handler errors")
	}
}

func TestAutoRespond_skipper(t *testing.T) {
	mw := AutoRespondWithConfig(AutoRespondConfig{
		Skipper: func(_ maxigobot.Context) bool { return true },
	})
	ctx := &mockContext{
		callback: &maxigo.Callback{CallbackID: "cb3"},
	}

	handler := mw(func(c maxigobot.Context) error {
		return nil
	})

	err := handler(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.respondCalled {
		t.Error("Respond should not be called when skipped")
	}
}

func TestAutoRespond_default(t *testing.T) {
	// AutoRespond() with defaults should not panic.
	mw := AutoRespond()
	handler := mw(func(c maxigobot.Context) error {
		return nil
	})

	err := handler(&mockContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
