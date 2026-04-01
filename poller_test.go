package maxigobot

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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

// newPollerTestBot creates a Bot with a client pointing at the given test server URL.
func newPollerTestBot(t *testing.T, serverURL string) *Bot {
	t.Helper()
	c, err := maxigo.New("test-token", maxigo.WithBaseURL(serverURL))
	if err != nil {
		t.Fatalf("maxigo.New: %v", err)
	}
	b, err := New("test-token", WithClient(c))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b
}

func TestLongPoller_Poll_errorRoutesToOnError(t *testing.T) {
	// Server always returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, `{"message":"internal error"}`)
	}))
	defer srv.Close()

	b := newPollerTestBot(t, srv.URL)

	var mu sync.Mutex
	var gotErr error
	b.OnError = func(err error, c Context) {
		mu.Lock()
		defer mu.Unlock()
		if gotErr == nil {
			gotErr = err
		}
	}

	updates := make(chan any, 10)
	stop := make(chan struct{})
	poller := &LongPoller{Timeout: 1}

	go poller.Poll(b, updates, stop)

	// Wait for at least one error to be captured.
	deadline := time.After(3 * time.Second)
	for {
		mu.Lock()
		captured := gotErr
		mu.Unlock()
		if captured != nil {
			break
		}
		select {
		case <-deadline:
			close(stop)
			t.Fatal("timed out waiting for OnError to be called")
		case <-time.After(10 * time.Millisecond):
		}
	}

	close(stop)
	// Drain updates channel to let Poll finish.
	for range updates {
	}

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotErr.Error(), "poll error") {
		t.Errorf("expected 'poll error' in message, got: %v", gotErr)
	}
}

func TestLongPoller_Poll_errorPreservesChain(t *testing.T) {
	// Server always returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, `{"message":"test"}`)
	}))
	defer srv.Close()

	b := newPollerTestBot(t, srv.URL)

	var mu sync.Mutex
	var gotErr error
	b.OnError = func(err error, c Context) {
		mu.Lock()
		defer mu.Unlock()
		if gotErr == nil {
			gotErr = err
		}
	}

	updates := make(chan any, 10)
	stop := make(chan struct{})
	poller := &LongPoller{Timeout: 1}

	go poller.Poll(b, updates, stop)

	deadline := time.After(3 * time.Second)
	for {
		mu.Lock()
		captured := gotErr
		mu.Unlock()
		if captured != nil {
			break
		}
		select {
		case <-deadline:
			close(stop)
			t.Fatal("timed out waiting for OnError")
		case <-time.After(10 * time.Millisecond):
		}
	}

	close(stop)
	for range updates {
	}

	mu.Lock()
	defer mu.Unlock()

	// %w wrapping should preserve the error chain.
	var apiErr *maxigo.Error
	if !errors.As(gotErr, &apiErr) {
		t.Errorf("expected errors.As to find *maxigo.Error, got: %T: %v", gotErr, gotErr)
	}
}

func TestLongPoller_Poll_parseErrorRoutesToOnError(t *testing.T) {
	// Return an update with valid JSON wrapper but invalid update body.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// updates contains a raw JSON that has a known update_type but invalid body.
		_, _ = fmt.Fprintln(w, `{"updates":[{"update_type":"message_created","message":"not-an-object"}],"marker":1}`)
	}))
	defer srv.Close()

	b := newPollerTestBot(t, srv.URL)

	var mu sync.Mutex
	var gotErr error
	b.OnError = func(err error, c Context) {
		mu.Lock()
		defer mu.Unlock()
		if gotErr == nil {
			gotErr = err
		}
	}

	updates := make(chan any, 10)
	stop := make(chan struct{})
	poller := &LongPoller{Timeout: 1}

	go poller.Poll(b, updates, stop)

	deadline := time.After(3 * time.Second)
	for {
		mu.Lock()
		captured := gotErr
		mu.Unlock()
		if captured != nil {
			break
		}
		select {
		case <-deadline:
			close(stop)
			t.Fatal("timed out waiting for OnError on parse error")
		case <-time.After(10 * time.Millisecond):
		}
	}

	close(stop)
	for range updates {
	}

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotErr.Error(), "parse update error") {
		t.Errorf("expected 'parse update error' in message, got: %v", gotErr)
	}
}

func TestLongPoller_Poll_panicRoutesToOnError(t *testing.T) {
	b, _ := New("test-token")
	// nil client causes a panic inside Poll when calling GetUpdates.
	b.client = nil

	var mu sync.Mutex
	var gotErr error
	b.OnError = func(err error, c Context) {
		mu.Lock()
		defer mu.Unlock()
		gotErr = err
	}

	updates := make(chan any, 10)
	stop := make(chan struct{})
	poller := &LongPoller{Timeout: 1}

	poller.Poll(b, updates, stop)

	mu.Lock()
	defer mu.Unlock()
	if gotErr == nil {
		t.Fatal("OnError should have been called after panic in poller")
	}
	if !strings.Contains(gotErr.Error(), "panic in poller") {
		t.Errorf("expected 'panic in poller' in message, got: %v", gotErr)
	}
}

func TestLongPoller_Poll_nilOnErrorFallback(t *testing.T) {
	// Server always returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, `{"message":"err"}`)
	}))
	defer srv.Close()

	b := newPollerTestBot(t, srv.URL)
	b.OnError = nil // Fallback to log.Printf — should not panic.

	updates := make(chan any, 10)
	stop := make(chan struct{})
	poller := &LongPoller{Timeout: 1}

	go poller.Poll(b, updates, stop)

	// Let it run briefly to ensure no panic.
	time.Sleep(100 * time.Millisecond)
	close(stop)
	for range updates {
	}
}
