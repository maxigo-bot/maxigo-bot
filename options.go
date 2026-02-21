package maxigobot

import (
	maxigo "github.com/maxigo-bot/maxigo-client"
)

// Option configures the Bot during creation.
type Option func(*Bot)

// WithLongPolling sets up long polling with the given timeout in seconds.
func WithLongPolling(timeout int) Option {
	return func(b *Bot) {
		b.poller = &LongPoller{Timeout: timeout}
	}
}

// WithClient injects a pre-configured maxigo-client (useful for testing).
func WithClient(c *maxigo.Client) Option {
	return func(b *Bot) {
		b.client = c
	}
}

// WithUpdateTypes filters which update types the poller will receive.
func WithUpdateTypes(types ...string) Option {
	return func(b *Bot) {
		b.updateTypes = types
	}
}

// sendConfig holds parameters for a send/reply/edit operation.
type sendConfig struct {
	ReplyTo            string
	Notify             *bool
	Format             *maxigo.TextFormat
	Attachments        []maxigo.AttachmentRequest
	DisableLinkPreview bool
}

// SendOption configures a send/reply/edit operation.
type SendOption func(*sendConfig)

// WithReplyTo sets the message ID to reply to.
func WithReplyTo(messageID string) SendOption {
	return func(cfg *sendConfig) {
		cfg.ReplyTo = messageID
	}
}

// WithNotify controls whether chat members are notified.
func WithNotify(notify bool) SendOption {
	return func(cfg *sendConfig) {
		cfg.Notify = &notify
	}
}

// WithFormat sets the text formatting mode (markdown or html).
func WithFormat(format maxigo.TextFormat) SendOption {
	return func(cfg *sendConfig) {
		cfg.Format = &format
	}
}

// WithAttachments adds attachments to the message.
func WithAttachments(attachments ...maxigo.AttachmentRequest) SendOption {
	return func(cfg *sendConfig) {
		cfg.Attachments = append(cfg.Attachments, attachments...)
	}
}

// WithDisableLinkPreview prevents the server from generating link previews.
func WithDisableLinkPreview() SendOption {
	return func(cfg *sendConfig) {
		cfg.DisableLinkPreview = true
	}
}

// WithKeyboard adds an inline keyboard to the message.
// Each argument is a row of buttons.
func WithKeyboard(rows ...[]maxigo.Button) SendOption {
	return WithAttachments(maxigo.NewInlineKeyboardAttachment(rows))
}

// buildSendConfig merges all send options into a sendConfig.
func buildSendConfig(opts []SendOption) sendConfig {
	var cfg sendConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// toMessageBody converts text + sendConfig into a maxigo NewMessageBody.
func toMessageBody(text string, cfg sendConfig) *maxigo.NewMessageBody {
	body := &maxigo.NewMessageBody{
		Text:               maxigo.Some(text),
		DisableLinkPreview: cfg.DisableLinkPreview,
	}

	if cfg.Notify != nil {
		body.Notify = maxigo.Some(*cfg.Notify)
	}
	if cfg.Format != nil {
		body.Format = maxigo.Some(*cfg.Format)
	}
	if len(cfg.Attachments) > 0 {
		body.Attachments = cfg.Attachments
	}
	if cfg.ReplyTo != "" {
		body.Link = &maxigo.NewMessageLink{
			Type: maxigo.LinkReply,
			MID:  cfg.ReplyTo,
		}
	}

	return body
}
