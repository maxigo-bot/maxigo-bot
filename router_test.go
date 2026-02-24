package maxigobot

import (
	"encoding/json"
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantCmd   string
		wantPl    string
		wantIsCmd bool
	}{
		{"simple command", "/start", "start", "", true},
		{"command with payload", "/start:hello", "start", "hello", true},
		{"payload with colons", "/start:a:b:c", "start", "a:b:c", true},
		{"command with empty payload", "/start:", "start", "", true},
		{"plain text", "hello world", "", "", false},
		{"empty text", "", "", "", false},
		{"slash only", "/", "", "", true},
		{"slash with colon", "/:payload", "", "payload", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, pl, isCmd := parseCommand(tt.text)
			if cmd != tt.wantCmd {
				t.Errorf("command = %q, want %q", cmd, tt.wantCmd)
			}
			if pl != tt.wantPl {
				t.Errorf("payload = %q, want %q", pl, tt.wantPl)
			}
			if isCmd != tt.wantIsCmd {
				t.Errorf("isCommand = %v, want %v", isCmd, tt.wantIsCmd)
			}
		})
	}
}

func TestResolveEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		update       any
		wantEndpoint string
		wantCmd      string
		wantPayload  string
	}{
		{
			"command message",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{Text: ptrString("/help")}},
			},
			"/help", "help", "",
		},
		{
			"command with payload",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{Text: ptrString("/start:hello")}},
			},
			"/start", "start", "hello",
		},
		{
			"plain text",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{Text: ptrString("hello")}},
			},
			OnText, "", "",
		},
		{
			"contact attachment with text",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{
					Text:        ptrString("John Doe"),
					Attachments: []json.RawMessage{json.RawMessage(`{"type":"contact"}`)},
				}},
			},
			OnContact, "", "",
		},
		{
			"contact attachment without text",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{
					Attachments: []json.RawMessage{json.RawMessage(`{"type":"contact"}`)},
				}},
			},
			OnContact, "", "",
		},
		{
			"photo attachment",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{
					Attachments: []json.RawMessage{json.RawMessage(`{"type":"image"}`)},
				}},
			},
			OnPhoto, "", "",
		},
		{
			"location attachment",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{
					Attachments: []json.RawMessage{json.RawMessage(`{"type":"location"}`)},
				}},
			},
			OnLocation, "", "",
		},
		{
			"unknown attachment falls through to text",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{
					Text:        ptrString("hello"),
					Attachments: []json.RawMessage{json.RawMessage(`{"type":"sticker"}`)},
				}},
			},
			OnText, "", "",
		},
		{
			"unknown attachment no text",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{
					Attachments: []json.RawMessage{json.RawMessage(`{"type":"sticker"}`)},
				}},
			},
			OnMessage, "", "",
		},
		{
			"malformed attachment JSON falls through to text",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{
					Text:        ptrString("hello"),
					Attachments: []json.RawMessage{json.RawMessage(`{invalid}`)},
				}},
			},
			OnText, "", "",
		},
		{
			"command with photo attachment routes to photo",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{
					Text:        ptrString("/start"),
					Attachments: []json.RawMessage{json.RawMessage(`{"type":"image"}`)},
				}},
			},
			OnPhoto, "", "",
		},
		{
			"no text",
			&maxigo.MessageCreatedUpdate{},
			OnMessage, "", "",
		},
		{
			"callback",
			&maxigo.MessageCallbackUpdate{
				Callback: maxigo.Callback{Payload: "confirm"},
			},
			OnCallback("confirm"), "", "",
		},
		{
			"edited message",
			&maxigo.MessageEditedUpdate{},
			OnEdited, "", "",
		},
		{
			"removed message",
			&maxigo.MessageRemovedUpdate{},
			OnRemoved, "", "",
		},
		{
			"bot started",
			&maxigo.BotStartedUpdate{},
			OnBotStarted, "", "",
		},
		{
			"bot stopped",
			&maxigo.BotStoppedUpdate{},
			OnBotStopped, "", "",
		},
		{
			"bot added",
			&maxigo.BotAddedUpdate{},
			OnBotAdded, "", "",
		},
		{
			"bot removed",
			&maxigo.BotRemovedUpdate{},
			OnBotRemoved, "", "",
		},
		{
			"user added",
			&maxigo.UserAddedUpdate{},
			OnUserAdded, "", "",
		},
		{
			"user removed",
			&maxigo.UserRemovedUpdate{},
			OnUserRemoved, "", "",
		},
		{
			"chat title changed",
			&maxigo.ChatTitleChangedUpdate{},
			OnChatTitleChanged, "", "",
		},
		{
			"chat created",
			&maxigo.MessageChatCreatedUpdate{},
			OnChatCreated, "", "",
		},
		{
			"dialog muted",
			&maxigo.DialogMutedUpdate{},
			OnDialogMuted, "", "",
		},
		{
			"dialog unmuted",
			&maxigo.DialogUnmutedUpdate{},
			OnDialogUnmuted, "", "",
		},
		{
			"dialog cleared",
			&maxigo.DialogClearedUpdate{},
			OnDialogCleared, "", "",
		},
		{
			"dialog removed",
			&maxigo.DialogRemovedUpdate{},
			OnDialogRemoved, "", "",
		},
		{
			"unknown type",
			"something",
			"", "", "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep, cmd, pl := resolveEndpoint(tt.update)
			if ep != tt.wantEndpoint {
				t.Errorf("endpoint = %q, want %q", ep, tt.wantEndpoint)
			}
			if cmd != tt.wantCmd {
				t.Errorf("command = %q, want %q", cmd, tt.wantCmd)
			}
			if pl != tt.wantPayload {
				t.Errorf("payload = %q, want %q", pl, tt.wantPayload)
			}
		})
	}
}

func TestFindHandler_fallback(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	// Register OnText as fallback.
	onTextHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers[OnText] = onTextHandler

	// A command that has no handler should fall back to OnText.
	entry, _ := b.findHandler("/unknown", &maxigo.MessageCreatedUpdate{})
	if entry != onTextHandler {
		t.Error("should fall back to OnText handler")
	}
}

func TestFindHandler_onMessageCatchAll(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	onMsgHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers[OnMessage] = onMsgHandler

	// OnText not registered — should fall back to OnMessage.
	entry, _ := b.findHandler(OnText, &maxigo.MessageCreatedUpdate{})
	if entry != onMsgHandler {
		t.Error("should fall back to OnMessage handler")
	}
}

func TestFindHandler_callbackFallback(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	catchAll := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers[OnCallback("")] = catchAll

	// Specific callback not registered — should fall back to catch-all.
	entry, _ := b.findHandler(OnCallback("unknown"), &maxigo.MessageCallbackUpdate{})
	if entry != catchAll {
		t.Error("should fall back to catch-all callback handler")
	}
}

func TestFindHandler_exactMatch(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	startHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	onTextHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers["/start"] = startHandler
	b.handlers[OnText] = onTextHandler

	// Exact command match should win over OnText.
	entry, _ := b.findHandler("/start", &maxigo.MessageCreatedUpdate{})
	if entry != startHandler {
		t.Error("should match exact /start handler, not OnText")
	}
}

func TestFindHandler_groupPriority(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	botHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers["/start"] = botHandler

	g := b.Group()
	groupHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	g.handlers["/start"] = groupHandler

	// Group handler should be found first.
	entry, _ := b.findHandler("/start", &maxigo.MessageCreatedUpdate{})
	if entry != groupHandler {
		t.Error("group handler should take priority over bot handler")
	}
}

func TestFindHandler_attachmentExactMatch(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	contactHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	onTextHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers[OnContact] = contactHandler
	b.handlers[OnText] = onTextHandler

	// OnContact registered — should match exactly, not fall back to OnText.
	entry, _ := b.findHandler(OnContact, &maxigo.MessageCreatedUpdate{})
	if entry != contactHandler {
		t.Error("should match exact OnContact handler")
	}
}

func TestFindHandler_attachmentFallbackToOnText(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	onTextHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers[OnText] = onTextHandler

	// OnContact not registered, message has text — should fall back to OnText.
	entry, _ := b.findHandler(OnContact, &maxigo.MessageCreatedUpdate{
		Message: maxigo.Message{Body: maxigo.MessageBody{Text: ptrString("John Doe")}},
	})
	if entry != onTextHandler {
		t.Error("should fall back to OnText when attachment handler not registered and message has text")
	}
}

func TestFindHandler_attachmentFallbackToOnMessage(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	onMsgHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers[OnMessage] = onMsgHandler

	// OnContact not registered, no OnText — should fall back to OnMessage.
	entry, _ := b.findHandler(OnContact, &maxigo.MessageCreatedUpdate{})
	if entry != onMsgHandler {
		t.Error("should fall back to OnMessage when no attachment handler and no OnText")
	}
}

func TestFindHandler_attachmentSkipsOnTextWhenNoText(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	onTextHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	onMsgHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	b.handlers[OnText] = onTextHandler
	b.handlers[OnMessage] = onMsgHandler

	// OnContact not registered, message has no text — should skip OnText, fall back to OnMessage.
	entry, _ := b.findHandler(OnContact, &maxigo.MessageCreatedUpdate{})
	if entry != onMsgHandler {
		t.Error("should skip OnText and fall back to OnMessage when message has no text")
	}
}

func ptrString(s string) *string { return &s }

func TestFindHandler_attachmentNoHandler(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	// Attachment endpoint with no handlers at all — should return nil.
	entry, _ := b.findHandler(OnContact, &maxigo.MessageCreatedUpdate{})
	if entry != nil {
		t.Error("should return nil when no handler matches attachment endpoint")
	}
}

func TestFindHandler_attachmentGroupFallbackToOnText(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	g := b.Group()
	groupTextHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	g.handlers[OnText] = groupTextHandler

	// OnPhoto not registered, message has text, group has OnText — should fall back to group OnText.
	entry, _ := b.findHandler(OnPhoto, &maxigo.MessageCreatedUpdate{
		Message: maxigo.Message{Body: maxigo.MessageBody{Text: ptrString("caption")}},
	})
	if entry != groupTextHandler {
		t.Error("should fall back to group OnText handler for attachment with text")
	}
}

func TestFindHandler_attachmentGroupFallbackToOnMessage(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	g := b.Group()
	groupMsgHandler := &handlerEntry{handler: func(c Context) error { return nil }}
	g.handlers[OnMessage] = groupMsgHandler

	// OnLocation not registered, no text, group has OnMessage — should fall back to group OnMessage.
	entry, _ := b.findHandler(OnLocation, &maxigo.MessageCreatedUpdate{})
	if entry != groupMsgHandler {
		t.Error("should fall back to group OnMessage handler for attachment")
	}
}

func TestFindHandler_noMatch(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	entry, _ := b.findHandler("/nothing", &maxigo.MessageCreatedUpdate{})
	if entry != nil {
		t.Error("should return nil when no handler matches")
	}
}
