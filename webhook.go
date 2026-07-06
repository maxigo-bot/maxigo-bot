package maxigobot

import (
	"crypto/subtle"
	"io"
	"net/http"
	"sync"
)

// WebhookSecretHeader is the HTTP header in which the Max Bot API sends back
// the secret provided at subscription time (Client.Subscribe).
const WebhookSecretHeader = "X-Max-Bot-Api-Secret"

const (
	defaultWebhookQueueSize   = 128
	defaultWebhookMaxBodySize = 1 << 20 // 1 MB
)

// WebhookPoller implements [Poller] by receiving updates over HTTP webhooks
// instead of long polling. It is also an [http.Handler]: mount it on your
// HTTPS server and subscribe the bot to the public URL.
//
//	wh := &maxigobot.WebhookPoller{Secret: "s3cret"}
//	b, err := maxigobot.New(token, maxigobot.WithPoller(wh))
//	// handle err
//	http.Handle("/webhook", wh)
//	go func() { _ = http.ListenAndServeTLS(":443", cert, key, nil) }()
//	// client.Subscribe(ctx, "https://example.com/webhook", nil, "s3cret")
//	b.Start()
//
// Since 2026-05-25 the Max Bot API delivers webhooks only to HTTPS endpoints
// with certificates from trusted CAs (including Минцифры); plain HTTP and
// self-signed certificates are rejected.
//
// Incoming requests are verified against Secret, parsed, and queued. If the
// queue is full or the bot is stopped, the handler replies 503 so that the
// Max Bot API redelivers the update later. Unknown update types are
// acknowledged with 200 and skipped, matching LongPoller behavior.
type WebhookPoller struct {
	// Secret is the expected value of the X-Max-Bot-Api-Secret header, as
	// passed to Client.Subscribe. The comparison is constant-time. An empty
	// Secret requires the header to be empty or absent; always set a
	// non-empty secret in production.
	Secret string
	// QueueSize is the capacity of the internal update queue between
	// ServeHTTP and Poll (default 128).
	QueueSize int
	// MaxBodySize limits the accepted request body size in bytes (default 1 MB).
	MaxBodySize int64

	initOnce sync.Once
	queue    chan any

	mu   sync.Mutex
	stop <-chan struct{} // set by Poll; nil until the poller starts
}

// init lazily creates the internal queue so that ServeHTTP can accept
// updates even before Poll starts.
func (p *WebhookPoller) init() {
	p.initOnce.Do(func() {
		size := p.QueueSize
		if size <= 0 {
			size = defaultWebhookQueueSize
		}
		p.queue = make(chan any, size)
	})
}

// Poll forwards queued webhook updates to the bot until stop is closed.
// It closes the updates channel before returning, as required by [Poller].
// The Bot argument is unused and may be nil.
func (p *WebhookPoller) Poll(_ *Bot, updates chan<- any, stop chan struct{}) {
	p.init()

	p.mu.Lock()
	p.stop = stop
	p.mu.Unlock()

	defer close(updates)
	for {
		select {
		case upd := <-p.queue:
			select {
			case updates <- upd:
			case <-stop:
				return
			}
		case <-stop:
			return
		}
	}
}

// ServeHTTP handles a webhook delivery from the Max Bot API.
func (p *WebhookPoller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.init()

	secret := r.Header.Get(WebhookSecretHeader)
	if subtle.ConstantTimeCompare([]byte(secret), []byte(p.Secret)) != 1 {
		http.Error(w, "invalid webhook secret", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maxBody := p.MaxBodySize
	if maxBody <= 0 {
		maxBody = defaultWebhookMaxBodySize
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	upd, err := ParseUpdate(body)
	if err != nil {
		http.Error(w, "failed to parse update", http.StatusBadRequest)
		return
	}
	if upd == nil {
		// Unknown update type: acknowledge so the server does not redeliver.
		w.WriteHeader(http.StatusOK)
		return
	}

	p.mu.Lock()
	stop := p.stop
	p.mu.Unlock()
	if stop != nil {
		select {
		case <-stop:
			http.Error(w, "bot is stopped", http.StatusServiceUnavailable)
			return
		default:
		}
	}

	select {
	case p.queue <- upd:
		w.WriteHeader(http.StatusOK)
	default:
		// Queue full: ask the Max Bot API to redeliver later.
		http.Error(w, "update queue is full", http.StatusServiceUnavailable)
	}
}
