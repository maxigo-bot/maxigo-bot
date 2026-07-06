package maxigobot

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

const webhookUpdateJSON = `{
	"update_type": "message_created",
	"timestamp": 1000,
	"message": {
		"sender": {"user_id": 1, "first_name": "Test", "is_bot": false, "last_activity_time": 0},
		"recipient": {"chat_id": 2, "chat_type": "dialog"},
		"timestamp": 1000,
		"body": {"mid": "m1", "seq": 1, "text": "hello"}
	}
}`

// postWebhook sends a POST request with the given body and secret header.
func postWebhook(t *testing.T, p *WebhookPoller, body, secret string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	if secret != "" {
		req.Header.Set(WebhookSecretHeader, secret)
	}
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)
	return rec
}

// startPoll runs p.Poll in a goroutine and returns the updates channel and a
// stop function that halts the poller and waits for the channel to close.
func startPoll(t *testing.T, p *WebhookPoller) (chan any, func()) {
	t.Helper()
	updates := make(chan any, 10)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		p.Poll(nil, updates, stop)
		close(done)
	}()
	return updates, func() {
		close(stop)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("Poll did not return after stop")
		}
	}
}

func TestWebhookPoller_DeliversUpdate(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret"}
	updates, stopPoll := startPoll(t, p)

	rec := postWebhook(t, p, webhookUpdateJSON, "s3cret")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	select {
	case upd := <-updates:
		msg, ok := upd.(*maxigo.MessageCreatedUpdate)
		if !ok {
			t.Fatalf("update type = %T, want *maxigo.MessageCreatedUpdate", upd)
		}
		if msg.Message.Body.Text == nil || *msg.Message.Body.Text != "hello" {
			t.Errorf("text = %v, want %q", msg.Message.Body.Text, "hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("update was not delivered")
	}

	stopPoll()
	if _, open := <-updates; open {
		t.Error("updates channel is not closed after stop")
	}
}

func TestWebhookPoller_RejectsWrongSecret(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret"}

	if rec := postWebhook(t, p, webhookUpdateJSON, "wrong"); rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong secret: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec := postWebhook(t, p, webhookUpdateJSON, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("missing secret: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestWebhookPoller_EmptySecretAllowsEmptyHeader(t *testing.T) {
	p := &WebhookPoller{}

	if rec := postWebhook(t, p, webhookUpdateJSON, ""); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec := postWebhook(t, p, webhookUpdateJSON, "unexpected"); rec.Code != http.StatusUnauthorized {
		t.Errorf("unexpected header: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestWebhookPoller_RejectsNonPOST(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret"}

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	req.Header.Set(WebhookSecretHeader, "s3cret")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestWebhookPoller_RejectsBadJSON(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret"}

	if rec := postWebhook(t, p, "{not json", "s3cret"); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestWebhookPoller_AcksUnknownUpdateType(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret"}
	updates, stopPoll := startPoll(t, p)
	defer stopPoll()

	rec := postWebhook(t, p, `{"update_type":"brand_new_type","timestamp":1}`, "s3cret")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (unknown types must be acked)", rec.Code, http.StatusOK)
	}

	select {
	case upd := <-updates:
		t.Errorf("unexpected update delivered: %T", upd)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestWebhookPoller_RejectsOversizedBody(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret", MaxBodySize: 10}

	if rec := postWebhook(t, p, webhookUpdateJSON, "s3cret"); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestWebhookPoller_StopWhileForwarding(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret"}

	// Nobody reads from updates, so Poll blocks forwarding the update.
	updates := make(chan any)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		p.Poll(nil, updates, stop)
		close(done)
	}()

	if rec := postWebhook(t, p, webhookUpdateJSON, "s3cret"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Wait until Poll has taken the update from the queue and is blocked
	// on the unbuffered updates channel.
	deadline := time.Now().Add(2 * time.Second)
	for len(p.queue) != 0 {
		if time.Now().After(deadline) {
			t.Fatal("Poll did not pick up the queued update")
		}
		time.Sleep(time.Millisecond)
	}

	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Poll did not return after stop while forwarding")
	}
}

func TestWebhookPoller_QueueFullReturns503(t *testing.T) {
	// Poll is intentionally not running, so the queue does not drain.
	p := &WebhookPoller{Secret: "s3cret", QueueSize: 1}

	if rec := postWebhook(t, p, webhookUpdateJSON, "s3cret"); rec.Code != http.StatusOK {
		t.Fatalf("first request: status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec := postWebhook(t, p, webhookUpdateJSON, "s3cret"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("queue full: status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestWebhookPoller_After503QueueDrainDelivers(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret", QueueSize: 1}

	// Fill the queue before the poller starts.
	if rec := postWebhook(t, p, webhookUpdateJSON, "s3cret"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	updates, stopPoll := startPoll(t, p)
	defer stopPoll()

	select {
	case <-updates:
	case <-time.After(2 * time.Second):
		t.Fatal("queued update was not delivered after Poll started")
	}
}

func TestWebhookPoller_ServesAfterStopWith503(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret"}
	_, stopPoll := startPoll(t, p)
	stopPoll()

	if rec := postWebhook(t, p, webhookUpdateJSON, "s3cret"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status after stop = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestWithPoller(t *testing.T) {
	p := &WebhookPoller{Secret: "s3cret"}
	b, err := New("token", WithPoller(p))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if b.poller != p {
		t.Errorf("poller = %v, want the WebhookPoller instance", b.poller)
	}
}
