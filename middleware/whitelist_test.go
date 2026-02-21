package middleware

import (
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
	maxigobot "github.com/maxigo-bot/maxigo-bot"
)

func TestWhitelist_allowed(t *testing.T) {
	called := false
	mw := Whitelist(1, 2, 3)
	handler := mw(func(c maxigobot.Context) error {
		called = true
		return nil
	})

	err := handler(&mockContext{sender: &maxigo.User{UserID: 2}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler should be called for whitelisted user")
	}
}

func TestWhitelist_blocked(t *testing.T) {
	called := false
	mw := Whitelist(1, 2, 3)
	handler := mw(func(c maxigobot.Context) error {
		called = true
		return nil
	})

	err := handler(&mockContext{sender: &maxigo.User{UserID: 999}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("handler should not be called for non-whitelisted user")
	}
}

func TestWhitelist_nilSender(t *testing.T) {
	called := false
	mw := Whitelist(1, 2)
	handler := mw(func(c maxigobot.Context) error {
		called = true
		return nil
	})

	err := handler(&mockContext{sender: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("handler should not be called when sender is nil")
	}
}

func TestBlacklist_allowed(t *testing.T) {
	called := false
	mw := Blacklist(10, 20)
	handler := mw(func(c maxigobot.Context) error {
		called = true
		return nil
	})

	err := handler(&mockContext{sender: &maxigo.User{UserID: 5}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler should be called for non-blacklisted user")
	}
}

func TestBlacklist_blocked(t *testing.T) {
	called := false
	mw := Blacklist(10, 20)
	handler := mw(func(c maxigobot.Context) error {
		called = true
		return nil
	})

	err := handler(&mockContext{sender: &maxigo.User{UserID: 10}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("handler should not be called for blacklisted user")
	}
}

func TestBlacklist_nilSender(t *testing.T) {
	called := false
	mw := Blacklist(10, 20)
	handler := mw(func(c maxigobot.Context) error {
		called = true
		return nil
	})

	err := handler(&mockContext{sender: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler should be called when sender is nil (pass-through)")
	}
}

func TestWhitelist_propagatesError(t *testing.T) {
	mw := Whitelist(1)
	handler := mw(func(c maxigobot.Context) error {
		return errForTest("handler error")
	})

	err := handler(&mockContext{sender: &maxigo.User{UserID: 1}})
	if err == nil || err.Error() != "handler error" {
		t.Errorf("error = %v, want 'handler error'", err)
	}
}
