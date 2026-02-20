package maxigobot

import (
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
	text := func(s string) *string { return &s }

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
				Message: maxigo.Message{Body: maxigo.MessageBody{Text: text("/help")}},
			},
			"/help", "help", "",
		},
		{
			"command with payload",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{Text: text("/start:hello")}},
			},
			"/start", "start", "hello",
		},
		{
			"plain text",
			&maxigo.MessageCreatedUpdate{
				Message: maxigo.Message{Body: maxigo.MessageBody{Text: text("hello")}},
			},
			OnText, "", "",
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

func TestFindHandler_noMatch(t *testing.T) {
	b := &Bot{handlers: make(map[string]*handlerEntry)}

	entry, _ := b.findHandler("/nothing", &maxigo.MessageCreatedUpdate{})
	if entry != nil {
		t.Error("should return nil when no handler matches")
	}
}
