package middleware

import (
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
	maxigobot "github.com/maxigo-bot/maxigo-bot"
)

func TestStripBotMention_groupStripsMention(t *testing.T) {
	text := "@bot_id hello"
	msg := &maxigo.Message{
		Recipient: maxigo.Recipient{ChatType: maxigo.ChatGroup},
		Body:      maxigo.MessageBody{Text: &text},
	}

	mw := StripBotMention()
	handler := mw(func(c maxigobot.Context) error { return nil })

	if err := handler(&mockContext{message: msg}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *msg.Body.Text != "hello" {
		t.Errorf("text = %q, want %q", *msg.Body.Text, "hello")
	}
}

func TestStripBotMention_dialogUntouched(t *testing.T) {
	text := "@someone hello"
	msg := &maxigo.Message{
		Recipient: maxigo.Recipient{ChatType: maxigo.ChatDialog},
		Body:      maxigo.MessageBody{Text: &text},
	}

	mw := StripBotMention()
	handler := mw(func(c maxigobot.Context) error { return nil })

	if err := handler(&mockContext{message: msg}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *msg.Body.Text != "@someone hello" {
		t.Errorf("dialog text was modified: %q", *msg.Body.Text)
	}
}

func TestStripBotMention_nilMessage(t *testing.T) {
	mw := StripBotMention()
	called := false
	handler := mw(func(c maxigobot.Context) error {
		called = true
		return nil
	})

	if err := handler(&mockContext{message: nil}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler should be called when message is nil")
	}
}

func TestStripBotMention_nilText(t *testing.T) {
	msg := &maxigo.Message{
		Recipient: maxigo.Recipient{ChatType: maxigo.ChatGroup},
		Body:      maxigo.MessageBody{Text: nil},
	}

	mw := StripBotMention()
	handler := mw(func(c maxigobot.Context) error { return nil })

	if err := handler(&mockContext{message: msg}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Body.Text != nil {
		t.Errorf("text should remain nil")
	}
}

func TestStripBotMention_propagatesError(t *testing.T) {
	text := "@bot_id /cmd"
	msg := &maxigo.Message{
		Recipient: maxigo.Recipient{ChatType: maxigo.ChatGroup},
		Body:      maxigo.MessageBody{Text: &text},
	}

	mw := StripBotMention()
	handler := mw(func(c maxigobot.Context) error {
		return errForTest("handler error")
	})

	err := handler(&mockContext{message: msg})
	if err == nil || err.Error() != "handler error" {
		t.Errorf("error = %v, want 'handler error'", err)
	}
}
