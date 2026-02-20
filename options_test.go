package maxigobot

import (
	"testing"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

func TestBuildSendConfig_defaults(t *testing.T) {
	cfg := buildSendConfig(nil)
	if cfg.ReplyTo != "" {
		t.Error("ReplyTo should be empty by default")
	}
	if cfg.Notify != nil {
		t.Error("Notify should be nil by default")
	}
	if cfg.Format != nil {
		t.Error("Format should be nil by default")
	}
}

func TestBuildSendConfig_allOptions(t *testing.T) {
	cfg := buildSendConfig([]SendOption{
		WithReplyTo("mid123"),
		WithNotify(false),
		WithFormat(maxigo.FormatMarkdown),
		WithDisableLinkPreview(),
		WithAttachments(maxigo.NewLocationAttachment(55.0, 37.0)),
	})

	if cfg.ReplyTo != "mid123" {
		t.Errorf("ReplyTo = %q, want %q", cfg.ReplyTo, "mid123")
	}
	if cfg.Notify == nil || *cfg.Notify != false {
		t.Error("Notify should be false")
	}
	if cfg.Format == nil || *cfg.Format != maxigo.FormatMarkdown {
		t.Error("Format should be markdown")
	}
	if !cfg.DisableLinkPreview {
		t.Error("DisableLinkPreview should be true")
	}
	if len(cfg.Attachments) != 1 {
		t.Errorf("Attachments count = %d, want 1", len(cfg.Attachments))
	}
}

func TestToMessageBody(t *testing.T) {
	cfg := sendConfig{
		ReplyTo: "mid456",
		Format:  ptr(maxigo.FormatHTML),
	}

	body := toMessageBody("hello", cfg)

	if !body.Text.Set || body.Text.Value != "hello" {
		t.Error("Text should be 'hello'")
	}
	if body.Link == nil {
		t.Fatal("Link should not be nil")
	}
	if body.Link.Type != maxigo.LinkReply {
		t.Errorf("Link.Type = %q, want %q", body.Link.Type, maxigo.LinkReply)
	}
	if body.Link.MID != "mid456" {
		t.Errorf("Link.MID = %q, want %q", body.Link.MID, "mid456")
	}
	if !body.Format.Set || body.Format.Value != maxigo.FormatHTML {
		t.Error("Format should be html")
	}
}

func TestToMessageBody_noReply(t *testing.T) {
	body := toMessageBody("test", sendConfig{})
	if body.Link != nil {
		t.Error("Link should be nil when no reply")
	}
}

func ptr[T any](v T) *T { return &v }
