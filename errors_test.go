package maxigobot

import (
	"errors"
	"io"
	"testing"
)

func TestBotError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *BotError
		wantMsg  string
	}{
		{
			"with endpoint",
			&BotError{Endpoint: "/start", Err: io.EOF},
			`maxigobot: handler "/start": EOF`,
		},
		{
			"without endpoint",
			&BotError{Err: io.EOF},
			"maxigobot: EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestBotError_Unwrap(t *testing.T) {
	inner := io.EOF
	err := &BotError{Endpoint: "/test", Err: inner}

	if !errors.Is(err, io.EOF) {
		t.Error("errors.Is should match inner error")
	}

	var botErr *BotError
	if !errors.As(err, &botErr) {
		t.Error("errors.As should match *BotError")
	}
	if botErr.Endpoint != "/test" {
		t.Errorf("Endpoint = %q, want %q", botErr.Endpoint, "/test")
	}
}
