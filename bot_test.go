package maxigobot

import (
	"errors"
	"strings"
	"testing"
	"time"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

func TestNew_emptyToken(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestNew_defaults(t *testing.T) {
	b, err := New("test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.client == nil {
		t.Error("client should not be nil")
	}
	if b.poller == nil {
		t.Error("poller should not be nil")
	}
	if _, ok := b.poller.(*LongPoller); !ok {
		t.Error("default poller should be LongPoller")
	}
}

func TestNew_withClient(t *testing.T) {
	c, _ := maxigo.New("injected-token")
	b, err := New("token", WithClient(c))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.client != c {
		t.Error("client should be the injected one")
	}
}

func TestNew_withLongPolling(t *testing.T) {
	b, err := New("token", WithLongPolling(60))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lp, ok := b.poller.(*LongPoller)
	if !ok {
		t.Fatal("poller should be LongPoller")
	}
	if lp.Timeout != 60 {
		t.Errorf("Timeout = %d, want 60", lp.Timeout)
	}
}

func TestNew_withUpdateTypes(t *testing.T) {
	b, err := New("token", WithUpdateTypes("message_created", "bot_started"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lp := b.poller.(*LongPoller)
	if len(lp.UpdateTypes) != 2 {
		t.Errorf("UpdateTypes count = %d, want 2", len(lp.UpdateTypes))
	}
}

func TestBot_Client(t *testing.T) {
	c, _ := maxigo.New("test-token")
	b, _ := New("token", WithClient(c))
	if b.Client() != c {
		t.Error("Client() should return the injected client")
	}
}

func TestBot_Handle(t *testing.T) {
	b, _ := New("token")
	b.Handle("/start", func(c Context) error { return nil })

	if _, ok := b.handlers["/start"]; !ok {
		t.Error("/start handler not registered")
	}
}

func TestBot_PreUse(t *testing.T) {
	b, _ := New("token")

	mw := func(next HandlerFunc) HandlerFunc { return next }
	b.Pre(mw)
	b.Use(mw, mw)

	if len(b.preMiddleware) != 1 {
		t.Errorf("pre middleware count = %d, want 1", len(b.preMiddleware))
	}
	if len(b.useMiddleware) != 2 {
		t.Errorf("use middleware count = %d, want 2", len(b.useMiddleware))
	}
}

func TestBot_Group(t *testing.T) {
	b, _ := New("token")
	g := b.Group()
	if g == nil {
		t.Fatal("group should not be nil")
	}
	if g.bot != b {
		t.Error("group.bot should reference parent bot")
	}
	if len(b.groups) != 1 {
		t.Errorf("groups count = %d, want 1", len(b.groups))
	}
}

func TestBot_ProcessUpdate_command(t *testing.T) {
	b, _ := New("token")

	var gotCmd, gotPayload string
	b.Handle("/start", func(c Context) error {
		gotCmd = c.Command()
		gotPayload = c.Payload()
		return nil
	})

	text := "/start:hello"
	upd := &maxigo.MessageCreatedUpdate{
		Message: maxigo.Message{Body: maxigo.MessageBody{Text: &text}},
	}
	b.processUpdate(upd)

	if gotCmd != "start" {
		t.Errorf("Command() = %q, want %q", gotCmd, "start")
	}
	if gotPayload != "hello" {
		t.Errorf("Payload() = %q, want %q", gotPayload, "hello")
	}
}

func TestBot_ProcessUpdate_lifecycle(t *testing.T) {
	b, _ := New("token")

	called := false
	b.Handle(OnBotStarted, func(c Context) error {
		called = true
		return nil
	})

	b.processUpdate(&maxigo.BotStartedUpdate{ChatID: 1, User: maxigo.User{UserID: 2}})

	if !called {
		t.Error("OnBotStarted handler was not called")
	}
}

func TestBot_ProcessUpdate_middlewareOrder(t *testing.T) {
	b, _ := New("token")

	var order []string
	b.Pre(func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			order = append(order, "pre")
			return next(c)
		}
	})
	b.Use(func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			order = append(order, "use")
			return next(c)
		}
	})
	b.Handle(OnBotStarted, func(c Context) error {
		order = append(order, "handler")
		return nil
	})

	b.processUpdate(&maxigo.BotStartedUpdate{})

	expected := []string{"pre", "use", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestBot_ProcessUpdate_groupMiddleware(t *testing.T) {
	b, _ := New("token")

	var order []string
	g := b.Group()
	g.Use(func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			order = append(order, "group")
			return next(c)
		}
	})
	g.Handle("/test", func(c Context) error {
		order = append(order, "handler")
		return nil
	})

	text := "/test"
	b.processUpdate(&maxigo.MessageCreatedUpdate{
		Message: maxigo.Message{Body: maxigo.MessageBody{Text: &text}},
	})

	expected := []string{"group", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestBot_ProcessUpdate_noHandler(t *testing.T) {
	b, _ := New("token")
	// Should not panic when no handler is registered.
	b.processUpdate(&maxigo.BotStartedUpdate{})
}

func TestBot_ProcessUpdate_onError(t *testing.T) {
	b, _ := New("token")

	expectedErr := errors.New("test error")
	var gotErr error
	b.OnError = func(err error, c Context) {
		gotErr = err
	}

	b.Handle(OnBotStarted, func(c Context) error {
		return expectedErr
	})

	b.processUpdate(&maxigo.BotStartedUpdate{})

	if gotErr == nil {
		t.Fatal("OnError should have been called")
	}
	if !errors.Is(gotErr, expectedErr) {
		t.Errorf("error = %v, want %v", gotErr, expectedErr)
	}
}

func TestBot_Stop(t *testing.T) {
	b, _ := New("token")
	// Stop should not panic and should close the channel.
	b.Stop()
	select {
	case <-b.stop:
		// OK, channel is closed.
	default:
		t.Error("stop channel should be closed")
	}
}

func TestBot_Stop_double(t *testing.T) {
	b, _ := New("token")
	// Double stop must not panic.
	b.Stop()
	b.Stop()
}

func TestBot_Stop_cancelsContext(t *testing.T) {
	b, _ := New("token")
	b.poller = &mockPoller{} // Empty poller, no updates.

	if b.ctx.Err() != nil {
		t.Fatal("context should not be cancelled before Stop")
	}

	done := make(chan struct{})
	go func() {
		b.Start()
		close(done)
	}()

	// Give Start time to begin polling.
	time.Sleep(20 * time.Millisecond)

	b.Stop()

	if b.ctx.Err() == nil {
		t.Fatal("context should be cancelled after Stop")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after Stop()")
	}
}

func TestBot_ProcessUpdate_panicRecovery(t *testing.T) {
	b, _ := New("token")

	var gotErr error
	b.OnError = func(err error, c Context) {
		gotErr = err
	}

	b.Handle(OnBotStarted, func(c Context) error {
		panic("test panic")
	})

	// Should not panic — recovery catches it.
	b.processUpdate(&maxigo.BotStartedUpdate{})

	if gotErr == nil {
		t.Fatal("OnError should have been called after panic")
	}
	if !strings.Contains(gotErr.Error(), "test panic") {
		t.Errorf("error should contain panic message, got: %v", gotErr)
	}
}

func TestBot_ProcessUpdate_panicRecovery_nilOnError(t *testing.T) {
	b, _ := New("token")
	b.OnError = nil

	b.Handle(OnBotStarted, func(c Context) error {
		panic("test panic")
	})

	// Should not panic even with nil OnError — logs to stderr instead.
	b.processUpdate(&maxigo.BotStartedUpdate{})
}

func TestBot_Start_stopsGracefully(t *testing.T) {
	b, _ := New("token")

	// Use a mock poller that sends one update then waits for stop.
	b.poller = &mockPoller{
		updates: []any{&maxigo.BotStartedUpdate{ChatID: 1, User: maxigo.User{UserID: 1}}},
	}

	called := false
	b.Handle(OnBotStarted, func(c Context) error {
		called = true
		return nil
	})

	done := make(chan struct{})
	go func() {
		b.Start()
		close(done)
	}()

	// Give Start time to process the update.
	time.Sleep(50 * time.Millisecond)
	b.Stop()

	select {
	case <-done:
		// Start returned — graceful shutdown works.
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after Stop()")
	}

	if !called {
		t.Error("handler should have been called")
	}
}

// mockPoller sends predefined updates, then waits for stop.
// It follows the Poller contract: closes updates before returning.
type mockPoller struct {
	updates []any
}

func (p *mockPoller) Poll(_ *Bot, updates chan<- any, stop chan struct{}) {
	defer close(updates)
	for _, u := range p.updates {
		updates <- u
	}
	<-stop
}
