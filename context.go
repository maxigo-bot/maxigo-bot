package maxigobot

import (
	gocontext "context"
	"errors"
	"strings"
	"sync"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

// Sentinel errors returned by Context methods.
var (
	ErrNoChatID   = errors.New("maxigobot: no chat ID available for this update")
	ErrNoMessage  = errors.New("maxigobot: no message available for this update")
	ErrNoCallback = errors.New("maxigobot: no callback available for this update")
	ErrNilPhoto   = errors.New("maxigobot: photo payload is required")
)

// Context provides handler access to the current update and bot API.
type Context interface {
	// Bot returns the parent Bot instance.
	Bot() *Bot
	// Update returns the raw update from maxigo-client.
	Update() maxigo.Update
	// API returns the underlying maxigo-client for direct API calls.
	API() *maxigo.Client
	// Ctx returns the request-scoped context.Context for cancellation and deadlines.
	Ctx() gocontext.Context

	// Sender returns the user who triggered the update (nil for some events).
	Sender() *maxigo.User
	// Chat returns the chat ID where the update occurred (0 if unavailable).
	Chat() int64

	// Message returns the original message (nil for lifecycle hooks).
	Message() *maxigo.Message
	// Text returns the full message text (empty if not a text message).
	Text() string
	// Command returns the command name without "/" (empty if not a command).
	Command() string
	// Payload returns the payload after ":" in commands or from BotStartedUpdate.
	Payload() string
	// Args returns the payload split by whitespace.
	Args() []string

	// Callback returns the callback object (nil if not a callback update).
	Callback() *maxigo.Callback
	// Data returns the callback payload string (empty if not a callback).
	Data() string

	// Send sends a message to the current chat.
	Send(text string, opts ...SendOption) error
	// Reply sends a reply to the current message.
	Reply(text string, opts ...SendOption) error
	// Edit edits the current message.
	Edit(text string, opts ...SendOption) error
	// Delete deletes the current message.
	Delete() error
	// SendPhoto sends a photo to the current chat.
	SendPhoto(photo *maxigo.PhotoAttachmentRequestPayload, opts ...SendOption) error

	// Respond answers a callback with a notification.
	Respond(text string) error
	// RespondAlert answers a callback with an alert.
	RespondAlert(text string) error

	// Notify sends a typing/action indicator to the current chat.
	Notify(action maxigo.SenderAction) error

	// Get retrieves a value from the context store.
	Get(key string) any
	// Set stores a value in the context store (thread-safe).
	Set(key string, val any)
}

// updateMeta holds pre-extracted common fields from an update.
// Computed once per update to avoid repeated type switches.
type updateMeta struct {
	base    maxigo.Update
	sender  *maxigo.User
	chatID  int64
	message *maxigo.Message
}

// extractMeta extracts common fields from a concrete update type.
func extractMeta(update any) updateMeta {
	var m updateMeta
	switch u := update.(type) {
	case *maxigo.MessageCreatedUpdate:
		m.base = u.Update
		m.sender = u.Message.Sender
		m.chatID = derefInt64(u.Message.Recipient.ChatID)
		m.message = &u.Message
	case *maxigo.MessageCallbackUpdate:
		m.base = u.Update
		m.sender = &u.Callback.User
		if u.Message != nil {
			m.chatID = derefInt64(u.Message.Recipient.ChatID)
		}
		m.message = u.Message
	case *maxigo.MessageEditedUpdate:
		m.base = u.Update
		m.sender = u.Message.Sender
		m.chatID = derefInt64(u.Message.Recipient.ChatID)
		m.message = &u.Message
	case *maxigo.MessageRemovedUpdate:
		m.base = u.Update
		m.chatID = u.ChatID
	case *maxigo.BotStartedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.BotStoppedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.BotAddedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.BotRemovedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.UserAddedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.UserRemovedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.ChatTitleChangedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.MessageChatCreatedUpdate:
		m.base = u.Update
		m.chatID = u.Chat.ChatID
	case *maxigo.DialogMutedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.DialogUnmutedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.DialogClearedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	case *maxigo.DialogRemovedUpdate:
		m.base = u.Update
		m.sender = &u.User
		m.chatID = u.ChatID
	}
	return m
}

// nativeContext is the default Context implementation.
type nativeContext struct {
	bot     *Bot
	update  any // concrete update type from maxigo-client
	meta    updateMeta
	ctx     gocontext.Context
	store   map[string]any
	storeMu sync.RWMutex
	command string
	payload string
}

func (c *nativeContext) Bot() *Bot             { return c.bot }
func (c *nativeContext) Update() maxigo.Update { return c.meta.base }
func (c *nativeContext) API() *maxigo.Client   { return c.bot.client }

func (c *nativeContext) Ctx() gocontext.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return gocontext.Background()
}

func (c *nativeContext) Sender() *maxigo.User    { return c.meta.sender }
func (c *nativeContext) Chat() int64             { return c.meta.chatID }
func (c *nativeContext) Message() *maxigo.Message { return c.meta.message }

func (c *nativeContext) Text() string {
	if msg := c.Message(); msg != nil && msg.Body.Text != nil {
		return *msg.Body.Text
	}
	return ""
}

func (c *nativeContext) Command() string { return c.command }

func (c *nativeContext) Payload() string {
	if c.payload != "" {
		return c.payload
	}
	// For BotStartedUpdate, return the start payload.
	if u, ok := c.update.(*maxigo.BotStartedUpdate); ok && u.Payload != nil {
		return *u.Payload
	}
	return ""
}

func (c *nativeContext) Args() []string {
	p := c.Payload()
	if p == "" {
		return nil
	}
	return strings.Fields(p)
}

func (c *nativeContext) Callback() *maxigo.Callback {
	if u, ok := c.update.(*maxigo.MessageCallbackUpdate); ok {
		return &u.Callback
	}
	return nil
}

func (c *nativeContext) Data() string {
	if cb := c.Callback(); cb != nil {
		return cb.Payload
	}
	return ""
}

func (c *nativeContext) Send(text string, opts ...SendOption) error {
	chatID := c.Chat()
	if chatID == 0 {
		return &BotError{Err: ErrNoChatID}
	}
	cfg := buildSendConfig(opts)
	body := toMessageBody(text, cfg)
	_, err := c.bot.client.SendMessage(c.Ctx(), chatID, body)
	return err
}

func (c *nativeContext) Reply(text string, opts ...SendOption) error {
	msg := c.Message()
	if msg == nil {
		return c.Send(text, opts...)
	}
	opts = append([]SendOption{WithReplyTo(msg.Body.MID)}, opts...)
	return c.Send(text, opts...)
}

func (c *nativeContext) Edit(text string, opts ...SendOption) error {
	msg := c.Message()
	if msg == nil {
		return &BotError{Err: ErrNoMessage}
	}
	cfg := buildSendConfig(opts)
	body := toMessageBody(text, cfg)
	_, err := c.bot.client.EditMessage(c.Ctx(), msg.Body.MID, body)
	return err
}

func (c *nativeContext) Delete() error {
	msg := c.Message()
	if msg == nil {
		return &BotError{Err: ErrNoMessage}
	}
	_, err := c.bot.client.DeleteMessage(c.Ctx(), msg.Body.MID)
	return err
}

func (c *nativeContext) SendPhoto(photo *maxigo.PhotoAttachmentRequestPayload, opts ...SendOption) error {
	if photo == nil {
		return &BotError{Err: ErrNilPhoto}
	}
	chatID := c.Chat()
	if chatID == 0 {
		return &BotError{Err: ErrNoChatID}
	}
	cfg := buildSendConfig(opts)
	cfg.Attachments = append(cfg.Attachments, maxigo.NewPhotoAttachment(*photo))
	body := toMessageBody("", cfg)
	_, err := c.bot.client.SendMessage(c.Ctx(), chatID, body)
	return err
}

func (c *nativeContext) Respond(text string) error {
	cb := c.Callback()
	if cb == nil {
		return &BotError{Err: ErrNoCallback}
	}
	_, err := c.bot.client.AnswerCallback(c.Ctx(), cb.CallbackID, &maxigo.CallbackAnswer{
		Notification: maxigo.Some(text),
	})
	return err
}

func (c *nativeContext) RespondAlert(text string) error {
	// Max Bot API uses the same AnswerCallback endpoint for notifications and alerts.
	return c.Respond(text)
}

func (c *nativeContext) Notify(action maxigo.SenderAction) error {
	chatID := c.Chat()
	if chatID == 0 {
		return &BotError{Err: ErrNoChatID}
	}
	_, err := c.bot.client.SendAction(c.Ctx(), chatID, action)
	return err
}

func (c *nativeContext) Get(key string) any {
	c.storeMu.RLock()
	defer c.storeMu.RUnlock()
	return c.store[key]
}

func (c *nativeContext) Set(key string, val any) {
	c.storeMu.Lock()
	defer c.storeMu.Unlock()
	if c.store == nil {
		c.store = make(map[string]any)
	}
	c.store[key] = val
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
