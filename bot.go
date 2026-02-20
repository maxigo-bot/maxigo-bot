package maxigobot

import (
	gocontext "context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

// ErrAlreadyStarted is returned when Start() is called on a bot that is already running.
var ErrAlreadyStarted = errors.New("maxigobot: bot is already started")

// Bot is the main framework entry point.
type Bot struct {
	client        *maxigo.Client
	poller        Poller
	handlers      map[string]*handlerEntry
	preMiddleware []MiddlewareFunc
	useMiddleware []MiddlewareFunc
	groups        []*Group
	updateTypes   []string
	stop          chan struct{}
	stopOnce      sync.Once
	wg            sync.WaitGroup
	ctx           gocontext.Context
	cancel        gocontext.CancelFunc
	started       atomic.Bool

	// OnError is called when a handler returns an error or a panic is recovered.
	// The Context argument may be nil when a panic is recovered before context is available.
	// If nil, errors are logged to stderr.
	OnError func(err error, c Context)
}

// New creates a new Bot with the given token and options.
func New(token string, opts ...Option) (*Bot, error) {
	if token == "" {
		return nil, errors.New("maxigobot: token is required")
	}

	ctx, cancel := gocontext.WithCancel(gocontext.Background())
	b := &Bot{
		handlers: make(map[string]*handlerEntry),
		stop:     make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}

	for _, opt := range opts {
		opt(b)
	}

	// Create client if not injected via WithClient.
	if b.client == nil {
		c, err := maxigo.New(token)
		if err != nil {
			return nil, err
		}
		b.client = c
	}

	// Default to long polling if no poller set.
	if b.poller == nil {
		b.poller = &LongPoller{Timeout: 30}
	}

	// Apply update types to poller if set.
	if len(b.updateTypes) > 0 {
		if lp, ok := b.poller.(*LongPoller); ok {
			lp.UpdateTypes = b.updateTypes
		}
	}

	return b, nil
}

// Client returns the underlying maxigo-client.
func (b *Bot) Client() *maxigo.Client {
	return b.client
}

// Pre appends middleware that runs before routing (all updates).
func (b *Bot) Pre(middleware ...MiddlewareFunc) {
	b.preMiddleware = append(b.preMiddleware, middleware...)
}

// Use appends middleware that runs after routing (matched handlers only).
func (b *Bot) Use(middleware ...MiddlewareFunc) {
	b.useMiddleware = append(b.useMiddleware, middleware...)
}

// Handle registers a handler for the given endpoint.
// Optional per-handler middleware is applied after global and group middleware.
func (b *Bot) Handle(endpoint any, h HandlerFunc, m ...MiddlewareFunc) {
	key := endpointKey(endpoint)
	b.handlers[key] = &handlerEntry{
		handler:    h,
		middleware: m,
	}
}

// Group creates a new handler group with an isolated middleware stack.
func (b *Bot) Group() *Group {
	g := &Group{
		bot:      b,
		handlers: make(map[string]*handlerEntry),
	}
	b.groups = append(b.groups, g)
	return g
}

// Start begins polling for updates and dispatching them to handlers.
// This method blocks until Stop() is called, the poller finishes,
// and all in-flight handlers complete.
// Panics if called more than once.
func (b *Bot) Start() {
	if !b.started.CompareAndSwap(false, true) {
		panic(ErrAlreadyStarted)
	}

	updates := make(chan any, 100)
	go b.poller.Poll(b, updates, b.stop)

	for upd := range updates {
		b.wg.Add(1)
		go func(u any) {
			defer b.wg.Done()
			b.processUpdate(u)
		}(upd)
	}

	b.wg.Wait()
}

// Stop signals the poller to stop and shuts down the bot.
// Safe to call multiple times.
func (b *Bot) Stop() {
	b.stopOnce.Do(func() {
		b.cancel()
		close(b.stop)
	})
}

// processUpdate routes a single update through middleware and to the matching handler.
func (b *Bot) processUpdate(update any) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic recovered: %v\n%s", r, debug.Stack())
			if b.OnError != nil {
				b.OnError(err, nil)
			} else {
				log.Printf("maxigobot: %v", err)
			}
		}
	}()

	endpoint, cmd, payload := resolveEndpoint(update)
	if endpoint == "" {
		return
	}

	ctx := &nativeContext{
		bot:     b,
		update:  update,
		meta:    extractMeta(update),
		command: cmd,
		payload: payload,
		ctx:     b.ctx,
	}

	// Pre-middleware runs on all updates.
	preHandler := HandlerFunc(func(c Context) error {
		entry, groupMW := b.findHandler(endpoint, update)
		if entry == nil {
			return nil // No handler registered — skip.
		}

		// Build handler chain: Use → Group → Per-handler → Handler.
		h := entry.handler
		h = applyMiddleware(h, entry.middleware...)
		h = applyMiddleware(h, groupMW...)
		h = applyMiddleware(h, b.useMiddleware...)

		return h(c)
	})

	chain := applyMiddleware(preHandler, b.preMiddleware...)

	if err := chain(ctx); err != nil {
		b.handleError(err, ctx, endpoint)
	}
}

func (b *Bot) handleError(err error, c Context, endpoint string) {
	if b.OnError != nil {
		b.OnError(err, c)
		return
	}
	log.Printf("maxigobot: handler %q error: %v", endpoint, err)
}
