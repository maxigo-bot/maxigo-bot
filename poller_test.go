package maxigobot

import (
	"encoding/json"
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

func TestParseUpdate_messageCreated(t *testing.T) {
	raw := json.RawMessage(`{
		"update_type": "message_created",
		"timestamp": 1000,
		"message": {
			"sender": {"user_id": 1, "first_name": "Test", "is_bot": false, "last_activity_time": 0},
			"recipient": {"chat_id": 2, "chat_type": "dialog"},
			"timestamp": 1000,
			"body": {"mid": "m1", "seq": 1, "text": "hello"}
		}
	}`)

	upd, err := ParseUpdate(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upd == nil {
		t.Fatal("expected non-nil update")
	}

	if msg, ok := upd.(*maxigo.MessageCreatedUpdate); !ok {
		t.Errorf("expected *maxigo.MessageCreatedUpdate, got %T", upd)
	} else if msg.Message.Body.MID != "m1" {
		t.Errorf("MID = %q, want %q", msg.Message.Body.MID, "m1")
	}
}

func TestParseUpdate_botStarted(t *testing.T) {
	raw := json.RawMessage(`{
		"update_type": "bot_started",
		"timestamp": 2000,
		"chat_id": 123,
		"user": {"user_id": 456, "first_name": "User", "is_bot": false, "last_activity_time": 0}
	}`)

	upd, err := ParseUpdate(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if upd == nil {
		t.Fatal("expected non-nil update")
	}
}

func TestParseUpdate_unknownType(t *testing.T) {
	raw := json.RawMessage(`{"update_type": "future_event", "timestamp": 3000}`)

	upd, err := ParseUpdate(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if upd != nil {
		t.Error("unknown update type should return nil")
	}
}

func TestParseUpdate_invalidJSON(t *testing.T) {
	raw := json.RawMessage(`{invalid`)

	_, err := ParseUpdate(raw)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseUpdate_allTypes(t *testing.T) {
	types := []string{
		"message_created", "message_callback", "message_edited", "message_removed",
		"bot_started", "bot_stopped", "bot_added", "bot_removed",
		"user_added", "user_removed", "chat_title_changed", "message_chat_created",
		"dialog_muted", "dialog_unmuted", "dialog_cleared", "dialog_removed",
	}

	for _, ut := range types {
		t.Run(ut, func(t *testing.T) {
			raw := json.RawMessage(`{"update_type":"` + ut + `","timestamp":1}`)
			upd, err := ParseUpdate(raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if upd == nil {
				t.Fatalf("expected non-nil update for type %q", ut)
			}
		})
	}
}

func TestLongPoller_defaultTimeout(t *testing.T) {
	lp := &LongPoller{}
	if lp.Timeout != 0 {
		t.Errorf("default Timeout = %d, want 0 (will use 30 in Poll)", lp.Timeout)
	}
}
