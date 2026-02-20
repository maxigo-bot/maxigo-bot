package maxigobot

import (
	gocontext "context"
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

func newTestBot() *Bot {
	c, _ := maxigo.New("test-token")
	return &Bot{
		client:   c,
		handlers: make(map[string]*handlerEntry),
	}
}

// newTestContext creates a nativeContext with pre-extracted meta for testing.
func newTestContext(bot *Bot, update any) *nativeContext {
	return &nativeContext{
		bot:    bot,
		update: update,
		meta:   extractMeta(update),
	}
}

func TestNativeContext_Sender(t *testing.T) {
	bot := newTestBot()
	user := maxigo.User{UserID: 123, FirstName: "Test"}

	tests := []struct {
		name   string
		update any
		want   int64
	}{
		{
			"message_created",
			&maxigo.MessageCreatedUpdate{Message: maxigo.Message{Sender: &user}},
			123,
		},
		{
			"callback",
			&maxigo.MessageCallbackUpdate{Callback: maxigo.Callback{User: user}},
			123,
		},
		{
			"bot_started",
			&maxigo.BotStartedUpdate{User: user},
			123,
		},
		{
			"dialog_muted",
			&maxigo.DialogMutedUpdate{User: user},
			123,
		},
		{
			"message_removed returns nil",
			&maxigo.MessageRemovedUpdate{},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext(bot, tt.update)
			sender := ctx.Sender()
			if tt.want == 0 {
				if sender != nil {
					t.Error("expected nil sender")
				}
				return
			}
			if sender == nil {
				t.Fatal("expected non-nil sender")
			}
			if sender.UserID != tt.want {
				t.Errorf("UserID = %d, want %d", sender.UserID, tt.want)
			}
		})
	}
}

func TestNativeContext_Chat(t *testing.T) {
	bot := newTestBot()
	chatID := int64(456)

	tests := []struct {
		name   string
		update any
		want   int64
	}{
		{
			"message_created",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}},
			},
			456,
		},
		{
			"bot_started",
			&maxigo.BotStartedUpdate{ChatID: 789},
			789,
		},
		{
			"message_removed",
			&maxigo.MessageRemovedUpdate{ChatID: 111},
			111,
		},
		{
			"nil chat_id in recipient",
			&maxigo.MessageCreatedUpdate{},
			0,
		},
		{
			"callback with message",
			&maxigo.MessageCallbackUpdate{
				Message: &maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}},
			},
			456,
		},
		{
			"callback without message",
			&maxigo.MessageCallbackUpdate{},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext(bot, tt.update)
			if got := ctx.Chat(); got != tt.want {
				t.Errorf("Chat() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNativeContext_Message(t *testing.T) {
	bot := newTestBot()

	t.Run("message_created", func(t *testing.T) {
		ctx := newTestContext(bot, &maxigo.MessageCreatedUpdate{Message: maxigo.Message{Body: maxigo.MessageBody{MID: "m1"}}})
		if msg := ctx.Message(); msg == nil || msg.Body.MID != "m1" {
			t.Error("expected message with MID=m1")
		}
	})

	t.Run("lifecycle returns nil", func(t *testing.T) {
		ctx := newTestContext(bot, &maxigo.BotStartedUpdate{})
		if msg := ctx.Message(); msg != nil {
			t.Error("expected nil message for lifecycle update")
		}
	})
}

func TestNativeContext_Text(t *testing.T) {
	bot := newTestBot()
	text := "hello world"

	ctx := newTestContext(bot, &maxigo.MessageCreatedUpdate{Message: maxigo.Message{Body: maxigo.MessageBody{Text: &text}}})
	if got := ctx.Text(); got != "hello world" {
		t.Errorf("Text() = %q, want %q", got, "hello world")
	}
}

func TestNativeContext_CommandPayload(t *testing.T) {
	bot := newTestBot()

	t.Run("from command", func(t *testing.T) {
		ctx := &nativeContext{bot: bot, command: "start", payload: "hello"}
		if ctx.Command() != "start" {
			t.Errorf("Command() = %q", ctx.Command())
		}
		if ctx.Payload() != "hello" {
			t.Errorf("Payload() = %q", ctx.Payload())
		}
	})

	t.Run("from bot_started", func(t *testing.T) {
		pl := "deep_link"
		ctx := &nativeContext{
			bot:    bot,
			update: &maxigo.BotStartedUpdate{Payload: &pl},
		}
		if ctx.Payload() != "deep_link" {
			t.Errorf("Payload() = %q, want %q", ctx.Payload(), "deep_link")
		}
	})

	t.Run("no payload", func(t *testing.T) {
		ctx := &nativeContext{bot: bot, update: &maxigo.BotStartedUpdate{}}
		if ctx.Payload() != "" {
			t.Errorf("Payload() = %q, want empty", ctx.Payload())
		}
	})
}

func TestNativeContext_Callback(t *testing.T) {
	bot := newTestBot()

	t.Run("callback update", func(t *testing.T) {
		ctx := &nativeContext{
			bot:    bot,
			update: &maxigo.MessageCallbackUpdate{Callback: maxigo.Callback{Payload: "btn1"}},
		}
		if cb := ctx.Callback(); cb == nil || cb.Payload != "btn1" {
			t.Error("expected callback with payload btn1")
		}
		if ctx.Data() != "btn1" {
			t.Errorf("Data() = %q", ctx.Data())
		}
	})

	t.Run("non-callback update", func(t *testing.T) {
		ctx := &nativeContext{bot: bot, update: &maxigo.MessageCreatedUpdate{}}
		if ctx.Callback() != nil {
			t.Error("expected nil callback")
		}
		if ctx.Data() != "" {
			t.Error("expected empty data")
		}
	})
}

func TestNativeContext_Store(t *testing.T) {
	ctx := &nativeContext{}

	// Get before any set.
	if v := ctx.Get("key"); v != nil {
		t.Error("expected nil for unset key")
	}

	ctx.Set("key", "value")
	if v := ctx.Get("key"); v != "value" {
		t.Errorf("Get('key') = %v, want 'value'", v)
	}

	ctx.Set("key", 42)
	if v := ctx.Get("key"); v != 42 {
		t.Errorf("Get('key') = %v, want 42", v)
	}
}

func TestNativeContext_Ctx_withContext(t *testing.T) {
	parent := gocontext.Background()
	child, cancel := gocontext.WithCancel(parent)
	defer cancel()

	ctx := &nativeContext{ctx: child}
	if ctx.Ctx() != child {
		t.Error("Ctx() should return the provided context")
	}
}

func TestNativeContext_Ctx_nilFallback(t *testing.T) {
	ctx := &nativeContext{}
	if ctx.Ctx() == nil {
		t.Error("Ctx() should return non-nil context even when not set")
	}
}

func TestNativeContext_Ctx_cancellation(t *testing.T) {
	parent, cancel := gocontext.WithCancel(gocontext.Background())
	ctx := &nativeContext{ctx: parent}

	if ctx.Ctx().Err() != nil {
		t.Fatal("context should not be cancelled yet")
	}

	cancel()

	if ctx.Ctx().Err() == nil {
		t.Fatal("context should be cancelled after cancel()")
	}
}
