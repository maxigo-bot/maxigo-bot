package maxigobot

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

// testBotWithServer creates a Bot backed by an httptest.Server.
func testBotWithServer(t *testing.T, handler http.HandlerFunc) *Bot {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := maxigo.New("test-token", maxigo.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("maxigo.New: %v", err)
	}

	b, err := New("test-token", WithClient(c))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b
}

func TestIntegration_SendMessage(t *testing.T) {
	var gotBody map[string]any

	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/messages" && r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			writeJSON(t, w, `{"message":{"sender":{"user_id":1,"first_name":"Bot","is_bot":true,"last_activity_time":0},"recipient":{"chat_id":100,"chat_type":"dialog"},"timestamp":1,"body":{"mid":"resp1","seq":1}}}`)
			return
		}
		w.WriteHeader(404)
	})

	chatID := int64(100)
	text := "hello"
	upd := &maxigo.MessageCreatedUpdate{Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}, Body: maxigo.MessageBody{MID: "m1", Text: &text}}}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Send("hello world")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotBody["text"] != "hello world" {
		t.Errorf("sent text = %v", gotBody["text"])
	}
}

func TestIntegration_Reply(t *testing.T) {
	var gotBody map[string]any

	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/messages" && r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			writeJSON(t, w, `{"message":{"sender":{"user_id":1,"first_name":"Bot","is_bot":true,"last_activity_time":0},"recipient":{"chat_id":100,"chat_type":"dialog"},"timestamp":1,"body":{"mid":"resp1","seq":1}}}`)
			return
		}
		w.WriteHeader(404)
	})

	chatID := int64(100)
	text := "original"
	upd := &maxigo.MessageCreatedUpdate{Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}, Body: maxigo.MessageBody{MID: "orig1", Text: &text}}}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Reply("reply text")
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}

	// Should have a link (reply) referencing the original message.
	link, ok := gotBody["link"].(map[string]any)
	if !ok {
		t.Fatal("expected link in body")
	}
	if link["type"] != "reply" {
		t.Errorf("link type = %v, want reply", link["type"])
	}
	if link["mid"] != "orig1" {
		t.Errorf("link mid = %v, want orig1", link["mid"])
	}
}

func TestIntegration_Edit(t *testing.T) {
	var gotPath string

	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			gotPath = r.URL.Path
			writeJSON(t, w, `{"success":true}`)
			return
		}
		w.WriteHeader(404)
	})

	chatID := int64(100)
	text := "old"
	upd := &maxigo.MessageCreatedUpdate{Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}, Body: maxigo.MessageBody{MID: "edit1", Text: &text}}}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Edit("new text")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if gotPath != "/messages" {
		t.Errorf("path = %q, want /messages", gotPath)
	}
}

func TestIntegration_Delete(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			writeJSON(t, w, `{"success":true}`)
			return
		}
		w.WriteHeader(404)
	})

	chatID := int64(100)
	upd := &maxigo.MessageCreatedUpdate{Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}, Body: maxigo.MessageBody{MID: "del1"}}}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Delete()
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestIntegration_Delete_noMessage(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	upd := &maxigo.BotStartedUpdate{}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Delete()
	if err == nil {
		t.Fatal("expected error for delete without message")
	}
}

func TestIntegration_Edit_noMessage(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	upd := &maxigo.BotStartedUpdate{}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Edit("text")
	if err == nil {
		t.Fatal("expected error for edit without message")
	}
}

func TestIntegration_Send_noChatID(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	// MessageCallbackUpdate without message has no chat ID.
	upd := &maxigo.MessageCallbackUpdate{}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Send("text")
	if err == nil {
		t.Fatal("expected error for send without chat ID")
	}
}

func TestIntegration_Respond(t *testing.T) {
	var gotPath string

	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(t, w, `{"success":true}`)
	})

	upd := &maxigo.MessageCallbackUpdate{
		Callback: maxigo.Callback{CallbackID: "cb123", Payload: "confirm"},
	}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Respond("OK!")
	if err != nil {
		t.Fatalf("Respond: %v", err)
	}
	if gotPath != "/answers" {
		t.Errorf("path = %q, want /answers", gotPath)
	}
}

func TestIntegration_Respond_noCallback(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	upd := &maxigo.MessageCreatedUpdate{}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Respond("text")
	if err == nil {
		t.Fatal("expected error for respond without callback")
	}
}

func TestIntegration_RespondAlert(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"success":true}`)
	})

	upd := &maxigo.MessageCallbackUpdate{
		Callback: maxigo.Callback{CallbackID: "cb1"},
	}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.RespondAlert("Alert!")
	if err != nil {
		t.Fatalf("RespondAlert: %v", err)
	}
}

func TestIntegration_Notify(t *testing.T) {
	var gotPath string

	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(t, w, `{"success":true}`)
	})

	chatID := int64(100)
	upd := &maxigo.MessageCreatedUpdate{Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}}}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Notify(maxigo.ActionTypingOn)
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if gotPath != "/chats/100/actions" {
		t.Errorf("path = %q, want /chats/100/actions", gotPath)
	}
}

func TestIntegration_Notify_noChatID(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	upd := &maxigo.MessageCallbackUpdate{}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Notify(maxigo.ActionTypingOn)
	if err == nil {
		t.Fatal("expected error for notify without chat ID")
	}
}

func TestIntegration_SendPhoto(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"message":{"sender":{"user_id":1,"first_name":"Bot","is_bot":true,"last_activity_time":0},"recipient":{"chat_id":100,"chat_type":"dialog"},"timestamp":1,"body":{"mid":"p1","seq":1}}}`)
	})

	chatID := int64(100)
	upd := &maxigo.MessageCreatedUpdate{Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}}}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	photo := &maxigo.PhotoAttachmentRequestPayload{
		URL: maxigo.Some("https://example.com/photo.jpg"),
	}
	err := ctx.SendPhoto(photo)
	if err != nil {
		t.Fatalf("SendPhoto: %v", err)
	}
}

func TestIntegration_SendPhoto_noChatID(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	upd := &maxigo.MessageCallbackUpdate{}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.SendPhoto(&maxigo.PhotoAttachmentRequestPayload{})
	if err == nil {
		t.Fatal("expected error for send photo without chat ID")
	}
}

func TestIntegration_Reply_noMessage(t *testing.T) {
	// Reply without message should fall back to Send.
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, `{"message":{"sender":{"user_id":1,"first_name":"Bot","is_bot":true,"last_activity_time":0},"recipient":{"chat_id":100,"chat_type":"dialog"},"timestamp":1,"body":{"mid":"r1","seq":1}}}`)
	})

	upd := &maxigo.BotStartedUpdate{ChatID: 100}
	ctx := &nativeContext{
		bot:    b,
		update: upd,
		meta:   extractMeta(upd),
	}

	err := ctx.Reply("text")
	if err != nil {
		t.Fatalf("Reply without message should fall back to Send: %v", err)
	}
}

func TestIntegration_EndToEnd(t *testing.T) {
	var mu sync.Mutex
	results := make(map[string]string)

	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/answers":
			writeJSON(t, w, `{"success":true}`)
		default:
			writeJSON(t, w, `{"message":{"sender":{"user_id":1,"first_name":"Bot","is_bot":true,"last_activity_time":0},"recipient":{"chat_id":100,"chat_type":"dialog"},"timestamp":1,"body":{"mid":"r1","seq":1}}}`)
		}
	})

	b.Handle("/start", func(c Context) error {
		mu.Lock()
		results["start"] = c.Payload()
		mu.Unlock()
		return c.Send("welcome")
	})

	b.Handle(OnText, func(c Context) error {
		mu.Lock()
		results["text"] = c.Text()
		mu.Unlock()
		return nil
	})

	b.Handle(OnBotStarted, func(c Context) error {
		mu.Lock()
		results["lifecycle"] = "bot_started"
		mu.Unlock()
		return nil
	})

	b.Handle(OnCallback("confirm"), func(c Context) error {
		mu.Lock()
		results["callback"] = c.Data()
		mu.Unlock()
		return c.Respond("ok")
	})

	var handlerErr error
	b.OnError = func(err error, c Context) {
		mu.Lock()
		handlerErr = err
		mu.Unlock()
	}

	// Simulate updates.
	chatID := int64(100)
	startText := "/start:hello"
	plainText := "just text"

	b.processUpdate(&maxigo.MessageCreatedUpdate{
		Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}, Body: maxigo.MessageBody{MID: "m1", Text: &startText}},
	})
	b.processUpdate(&maxigo.MessageCreatedUpdate{
		Message: maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}, Body: maxigo.MessageBody{MID: "m2", Text: &plainText}},
	})
	b.processUpdate(&maxigo.BotStartedUpdate{ChatID: 100, User: maxigo.User{UserID: 1}})
	b.processUpdate(&maxigo.MessageCallbackUpdate{
		Callback: maxigo.Callback{CallbackID: "cb1", Payload: "confirm"},
		Message:  &maxigo.Message{Recipient: maxigo.Recipient{ChatID: &chatID}},
	})

	mu.Lock()
	defer mu.Unlock()

	if results["start"] != "hello" {
		t.Errorf("start payload = %q, want %q", results["start"], "hello")
	}
	if results["text"] != "just text" {
		t.Errorf("text = %q, want %q", results["text"], "just text")
	}
	if results["lifecycle"] != "bot_started" {
		t.Errorf("lifecycle = %q, want %q", results["lifecycle"], "bot_started")
	}
	if results["callback"] != "confirm" {
		t.Errorf("callback = %q, want %q", results["callback"], "confirm")
	}
	if handlerErr != nil {
		t.Errorf("unexpected handler error: %v", handlerErr)
	}
}

func TestIntegration_ContextBotUpdateAPI(t *testing.T) {
	b := testBotWithServer(t, func(w http.ResponseWriter, r *http.Request) {})

	upd := maxigo.Update{UpdateType: maxigo.UpdateBotStarted, Timestamp: 1234}
	concreteUpd := &maxigo.BotStartedUpdate{Update: upd, ChatID: 1}
	ctx := &nativeContext{
		bot:    b,
		update: concreteUpd,
		meta:   extractMeta(concreteUpd),
	}

	if ctx.Bot() != b {
		t.Error("Bot() should return parent bot")
	}
	if ctx.Update().UpdateType != maxigo.UpdateBotStarted {
		t.Error("Update() should return base update")
	}
	if ctx.API() != b.client {
		t.Error("API() should return client")
	}
}

func TestIntegration_ProcessUpdate_allLifecycleTypes(t *testing.T) {
	// Verify processUpdate doesn't panic for all lifecycle types.
	b, _ := New("token")

	var called int
	handler := func(c Context) error {
		called++
		return nil
	}

	b.Handle(OnBotStarted, handler)
	b.Handle(OnBotStopped, handler)
	b.Handle(OnBotAdded, handler)
	b.Handle(OnBotRemoved, handler)
	b.Handle(OnUserAdded, handler)
	b.Handle(OnUserRemoved, handler)
	b.Handle(OnChatTitleChanged, handler)
	b.Handle(OnChatCreated, handler)
	b.Handle(OnDialogMuted, handler)
	b.Handle(OnDialogUnmuted, handler)
	b.Handle(OnDialogCleared, handler)
	b.Handle(OnDialogRemoved, handler)
	b.Handle(OnEdited, handler)
	b.Handle(OnRemoved, handler)

	updates := []any{
		&maxigo.BotStartedUpdate{},
		&maxigo.BotStoppedUpdate{},
		&maxigo.BotAddedUpdate{},
		&maxigo.BotRemovedUpdate{},
		&maxigo.UserAddedUpdate{},
		&maxigo.UserRemovedUpdate{},
		&maxigo.ChatTitleChangedUpdate{},
		&maxigo.MessageChatCreatedUpdate{},
		&maxigo.DialogMutedUpdate{},
		&maxigo.DialogUnmutedUpdate{},
		&maxigo.DialogClearedUpdate{},
		&maxigo.DialogRemovedUpdate{},
		&maxigo.MessageEditedUpdate{},
		&maxigo.MessageRemovedUpdate{},
	}

	for _, upd := range updates {
		b.processUpdate(upd)
	}

	if called != len(updates) {
		t.Errorf("called = %d, want %d", called, len(updates))
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
