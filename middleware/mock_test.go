package middleware

import (
	gocontext "context"

	maxigo "github.com/maxigo-bot/maxigo-client"
	maxigobot "github.com/maxigo-bot/maxigo-bot"
)

// mockContext implements maxigobot.Context for middleware testing.
type mockContext struct {
	update   maxigo.Update
	sender   *maxigo.User
	chatID   int64
	message  *maxigo.Message
	callback *maxigo.Callback
	text     string
	store    map[string]any

	// Tracking calls for assertions.
	respondCalled bool
	respondText   string
}

func (m *mockContext) Bot() *maxigobot.Bot       { return nil }
func (m *mockContext) Update() maxigo.Update      { return m.update }
func (m *mockContext) API() *maxigo.Client        { return nil }
func (m *mockContext) Ctx() gocontext.Context     { return gocontext.Background() }
func (m *mockContext) Sender() *maxigo.User       { return m.sender }
func (m *mockContext) Chat() int64                { return m.chatID }
func (m *mockContext) Message() *maxigo.Message   { return m.message }
func (m *mockContext) Text() string               { return m.text }
func (m *mockContext) Command() string            { return "" }
func (m *mockContext) Payload() string            { return "" }
func (m *mockContext) Args() []string             { return nil }
func (m *mockContext) Callback() *maxigo.Callback { return m.callback }
func (m *mockContext) Data() string {
	if m.callback != nil {
		return m.callback.Payload
	}
	return ""
}

func (m *mockContext) Send(_ string, _ ...maxigobot.SendOption) error      { return nil }
func (m *mockContext) Reply(_ string, _ ...maxigobot.SendOption) error     { return nil }
func (m *mockContext) Edit(_ string, _ ...maxigobot.SendOption) error      { return nil }
func (m *mockContext) Delete() error                                       { return nil }
func (m *mockContext) SendPhoto(_ *maxigo.PhotoAttachmentRequestPayload, _ ...maxigobot.SendOption) error {
	return nil
}

func (m *mockContext) Respond(text string) error {
	m.respondCalled = true
	m.respondText = text
	return nil
}
func (m *mockContext) RespondAlert(text string) error { return m.Respond(text) }
func (m *mockContext) Notify(_ maxigo.SenderAction) error { return nil }

func (m *mockContext) Get(key string) any {
	if m.store == nil {
		return nil
	}
	return m.store[key]
}

func (m *mockContext) Set(key string, val any) {
	if m.store == nil {
		m.store = make(map[string]any)
	}
	m.store[key] = val
}
