package maxigobot

import (
	"testing"
)

func TestGroup_Handle(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}
	g := b.Group()

	called := false
	g.Handle("/test", func(c Context) error {
		called = true
		return nil
	})

	if _, ok := g.handlers["/test"]; !ok {
		t.Fatal("handler not registered in group")
	}

	// Verify it's findable through the bot.
	entry, _ := b.findHandler("/test", nil)
	if entry == nil {
		t.Fatal("handler not found via bot.findHandler")
	}

	if err := entry.handler(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestGroup_Use(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}
	g := b.Group()

	var order []int
	g.Use(func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			order = append(order, 1)
			return next(c)
		}
	})

	g.Handle("/test", func(c Context) error {
		order = append(order, 2)
		return nil
	})

	entry, groupMW := b.findHandler("/test", nil)
	if entry == nil {
		t.Fatal("handler not found")
	}
	if len(groupMW) != 1 {
		t.Fatalf("group middleware count = %d, want 1", len(groupMW))
	}

	// Apply group middleware and run.
	h := applyMiddleware(entry.handler, groupMW...)
	if err := h(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("execution order = %v, want [1 2]", order)
	}
}

func TestGroup_HandleEvents(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}
	g := b.Group()

	g.Handle(OnBotStarted, func(c Context) error { return nil })
	g.Handle(OnCallback("confirm"), func(c Context) error { return nil })

	if _, ok := g.handlers[OnBotStarted]; !ok {
		t.Error("OnBotStarted not registered")
	}
	if _, ok := g.handlers[OnCallback("confirm")]; !ok {
		t.Error("OnCallback('confirm') not registered")
	}
}

func TestEndpointKey(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{"/start", "/start"},
		{OnText, OnText},
		{OnCallback("x"), "\fx"},
	}
	for _, tt := range tests {
		if got := endpointKey(tt.input); got != tt.want {
			t.Errorf("endpointKey(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEndpointKey_panicsOnInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-string endpoint")
		}
	}()
	endpointKey(42)
}
