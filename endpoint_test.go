package maxigobot

import (
	"testing"
)

func TestOnCallback(t *testing.T) {
	tests := []struct {
		name   string
		unique string
		want   string
	}{
		{"with payload", "confirm", "\fconfirm"},
		{"empty payload", "", "\f"},
		{"special chars", "btn:123", "\fbtn:123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OnCallback(tt.unique)
			if got != tt.want {
				t.Errorf("OnCallback(%q) = %q, want %q", tt.unique, got, tt.want)
			}
		})
	}
}

func TestEndpointConstants(t *testing.T) {
	// Verify all constants use \a prefix and don't clash.
	endpoints := []string{
		OnText, OnMessage, OnEdited, OnRemoved,
		OnBotStarted, OnBotStopped, OnBotAdded, OnBotRemoved,
		OnUserAdded, OnUserRemoved,
		OnChatTitleChanged, OnChatCreated,
		OnDialogMuted, OnDialogUnmuted, OnDialogCleared, OnDialogRemoved,
	}

	seen := make(map[string]bool, len(endpoints))
	for _, ep := range endpoints {
		if ep[0] != '\a' {
			t.Errorf("endpoint %q does not start with \\a", ep)
		}
		if seen[ep] {
			t.Errorf("duplicate endpoint %q", ep)
		}
		seen[ep] = true
	}
}
