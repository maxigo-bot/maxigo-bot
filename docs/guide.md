# maxigo-bot Guide

Gin-style bot framework for [Max messenger](https://max.ru). Router, middleware, context, groups — inspired by [Echo](https://echo.labstack.com) and [telebot](https://github.com/tucnak/telebot).

> **[Документация на русском](guide-ru.md)** | **[README](../README.md)**

## Installation

```bash
go get github.com/maxigo-bot/maxigo-bot
```

Requires Go 1.25+. Built on [maxigo-client](https://github.com/maxigo-bot/maxigo-client) — zero external transitive dependencies.

## Quick Start

```go
package main

import (
    "log"

    maxigobot "github.com/maxigo-bot/maxigo-bot"
)

func main() {
    b, err := maxigobot.New("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    b.Handle("/start", func(c maxigobot.Context) error {
        return c.Send("Hello, " + c.Sender().FirstName + "!")
    })

    b.Handle(maxigobot.OnText, func(c maxigobot.Context) error {
        return c.Reply("You said: " + c.Text())
    })

    b.Start() // blocks until Stop() is called
}
```

## Configuration

The bot is configured using functional options:

```go
b, err := maxigobot.New("TOKEN",
    maxigobot.WithLongPolling(30),              // long polling with 30s timeout (default)
    maxigobot.WithClient(preConfiguredClient),   // inject pre-configured maxigo-client
    maxigobot.WithUpdateTypes(                   // filter update types
        "message_created",
        "message_callback",
    ),
)
```

### Options

| Option | Description |
|--------|-------------|
| `WithLongPolling(timeout)` | Set long polling timeout in seconds (default: 30) |
| `WithClient(client)` | Inject a pre-configured `*maxigo.Client` (useful for testing) |
| `WithUpdateTypes(types...)` | Filter which update types the poller receives |

### Accessing the Client

If you need to make direct API calls:

```go
client := b.Client() // *maxigo.Client
bot, err := client.GetBot(ctx)
```

## Routing

### Commands

Max uses `:` as command separator (not space like Telegram): `/start:payload`.

```go
b.Handle("/start", func(c maxigobot.Context) error {
    name := c.Sender().FirstName
    payload := c.Payload() // text after ":"
    args := c.Args()       // payload split by whitespace
    return c.Send("Welcome, " + name + "!")
})

b.Handle("/help", func(c maxigobot.Context) error {
    return c.Send("Available commands: /start, /help")
})
```

### Events

```go
// Text messages (not commands)
b.Handle(maxigobot.OnText, func(c maxigobot.Context) error {
    return c.Reply("You said: " + c.Text())
})

// Any message (catch-all fallback for message_created)
b.Handle(maxigobot.OnMessage, func(c maxigobot.Context) error {
    return c.Send("Received a message")
})

// Lifecycle hooks
b.Handle(maxigobot.OnBotStarted, func(c maxigobot.Context) error {
    return c.Send("Welcome! Use /help to see available commands.")
})

b.Handle(maxigobot.OnBotAdded, func(c maxigobot.Context) error {
    return c.Send("Thanks for adding me!")
})

b.Handle(maxigobot.OnEdited, func(c maxigobot.Context) error {
    log.Printf("Message edited: %s", c.Text())
    return nil
})
```

### Callbacks

```go
// Send a message with inline keyboard
b.Handle("/menu", func(c maxigobot.Context) error {
    return c.Send("Choose an option:",
        maxigobot.WithAttachments(
            maxigo.NewInlineKeyboardAttachment([][]maxigo.Button{
                {
                    maxigo.NewCallbackButtonWithIntent("Confirm", "confirm", maxigo.IntentPositive),
                    maxigo.NewCallbackButtonWithIntent("Cancel", "cancel", maxigo.IntentNegative),
                },
            }),
        ),
    )
})

// Handle specific callback payload
b.Handle(maxigobot.OnCallback("confirm"), func(c maxigobot.Context) error {
    return c.Respond("Confirmed!")
})

b.Handle(maxigobot.OnCallback("cancel"), func(c maxigobot.Context) error {
    return c.Respond("Cancelled.")
})

// Catch-all callback handler (empty string matches any unmatched callback)
b.Handle(maxigobot.OnCallback(""), func(c maxigobot.Context) error {
    log.Printf("Unknown callback: %s", c.Data())
    return c.Respond("Unknown action")
})
```

### Fallback Chain

For `message_created` updates, routing tries handlers in this order:

1. **Exact command** (`/start`, `/help`, ...) — matches first
2. **`OnText`** — fallback for text messages (including unmatched commands)
3. **`OnMessage`** — catch-all for any message (photos, stickers, etc.)

For callbacks:

1. **Exact payload** (`OnCallback("confirm")`) — matches first
2. **`OnCallback("")`** — catch-all for unmatched callbacks

### Event Constants

| Constant | Update Type | Description |
|----------|-------------|-------------|
| `OnText` | `message_created` | Text message (not a command) |
| `OnMessage` | `message_created` | Any message (catch-all) |
| `OnEdited` | `message_edited` | Message edited |
| `OnRemoved` | `message_removed` | Message removed |
| `OnBotStarted` | `bot_started` | User pressed Start |
| `OnBotStopped` | `bot_stopped` | User stopped/blocked bot |
| `OnBotAdded` | `bot_added` | Bot added to chat |
| `OnBotRemoved` | `bot_removed` | Bot removed from chat |
| `OnUserAdded` | `user_added` | User added to chat |
| `OnUserRemoved` | `user_removed` | User removed from chat |
| `OnChatTitleChanged` | `chat_title_changed` | Chat title changed |
| `OnChatCreated` | `message_chat_created` | Chat created via button |
| `OnDialogMuted` | `dialog_muted` | User muted dialog |
| `OnDialogUnmuted` | `dialog_unmuted` | User unmuted dialog |
| `OnDialogCleared` | `dialog_cleared` | User cleared dialog history |
| `OnDialogRemoved` | `dialog_removed` | User removed dialog |
| `OnCallback("id")` | `message_callback` | Callback with specific payload |

## Middleware

Middleware wraps handlers to add cross-cutting behavior (logging, auth, recovery, etc.).

### Signature

```go
type HandlerFunc func(c Context) error
type MiddlewareFunc func(next HandlerFunc) HandlerFunc
```

### Two Levels

**Pre-middleware** — runs on ALL updates, before routing:

```go
b.Pre(recoverMiddleware)
b.Pre(requestIDMiddleware)
```

**Use-middleware** — runs only on matched handlers, after routing:

```go
b.Use(loggerMiddleware)
b.Use(authMiddleware)
```

### Execution Order

```
Update → Pre-middleware → Routing → Use-middleware → Group middleware → Per-handler middleware → Handler
```

If no handler matches, Use-middleware and beyond are skipped.

### Writing Middleware

```go
// Logger middleware — logs every matched update
func Logger() maxigobot.MiddlewareFunc {
    return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
        return func(c maxigobot.Context) error {
            start := time.Now()
            err := next(c)
            log.Printf("[%d] %s (%v)", c.Chat(), c.Text(), time.Since(start))
            return err
        }
    }
}

// Recover middleware — catches panics in handlers
func Recover() maxigobot.MiddlewareFunc {
    return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
        return func(c maxigobot.Context) (err error) {
            defer func() {
                if r := recover(); r != nil {
                    err = fmt.Errorf("panic: %v", r)
                }
            }()
            return next(c)
        }
    }
}

// Whitelist middleware — allows only specific users
func Whitelist(allowed ...int64) maxigobot.MiddlewareFunc {
    set := make(map[int64]bool, len(allowed))
    for _, id := range allowed {
        set[id] = true
    }
    return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
        return func(c maxigobot.Context) error {
            if sender := c.Sender(); sender != nil && set[sender.UserID] {
                return next(c)
            }
            return c.Send("Access denied.")
        }
    }
}
```

### Per-Handler Middleware

You can attach middleware to a specific handler:

```go
b.Handle("/admin", adminHandler, adminOnlyMiddleware, auditMiddleware)
```

These run after global and group middleware.

## Groups

Groups provide isolated middleware stacks for a subset of handlers:

```go
// Admin group — only admin users
admin := b.Group()
admin.Use(Whitelist(123456, 789012))
admin.Handle("/ban", banHandler)
admin.Handle("/mute", muteHandler)
admin.Handle("/stats", statsHandler)

// Public handlers — no admin middleware
b.Handle("/start", startHandler)
b.Handle("/help", helpHandler)
```

Groups inherit global `Use`-middleware but add their own on top:

```
Update → Pre → Routing → global Use → group middleware → per-handler middleware → Handler
```

## Context

`Context` provides handler access to the current update and bot API. A new context is created for each update.

### Update Data

```go
b.Handle("/info", func(c maxigobot.Context) error {
    // User who triggered the update (nil for some events)
    sender := c.Sender() // *maxigo.User

    // Chat ID where the update occurred (0 if unavailable)
    chatID := c.Chat() // int64

    // Original message (nil for lifecycle hooks like bot_started)
    msg := c.Message() // *maxigo.Message

    // Full message text (empty if not a text message)
    text := c.Text() // string

    // Raw update from maxigo-client
    upd := c.Update() // maxigo.Update

    // Request-scoped context.Context
    ctx := c.Ctx() // context.Context
})
```

### Commands and Payloads

```go
// For /greet:John Doe
b.Handle("/greet", func(c maxigobot.Context) error {
    c.Command() // "greet"
    c.Payload() // "John Doe"
    c.Args()    // ["John", "Doe"]
    return c.Send("Hello, " + c.Payload() + "!")
})

// BotStartedUpdate also has a payload
b.Handle(maxigobot.OnBotStarted, func(c maxigobot.Context) error {
    ref := c.Payload() // start payload (deep link)
    return c.Send("Welcome! Ref: " + ref)
})
```

### Callbacks

```go
b.Handle(maxigobot.OnCallback("action"), func(c maxigobot.Context) error {
    cb := c.Callback() // *maxigo.Callback
    data := c.Data()   // callback payload string

    // Respond with a notification
    return c.Respond("Action received: " + data)
})
```

### Sending Messages

```go
// Send to current chat
c.Send("Hello!")

// Reply to the current message
c.Reply("Got it!")

// Edit the current message
c.Edit("Updated text")

// Delete the current message
c.Delete()

// Send a photo
c.SendPhoto(&maxigo.PhotoAttachmentRequestPayload{
    Photos: tokens.Photos,
})

// Send typing indicator
c.Notify(maxigo.ActionTypingOn)
```

### Send Options

```go
c.Send("Hello!",
    maxigobot.WithReplyTo(messageID),                    // reply to specific message
    maxigobot.WithNotify(false),                          // disable notification
    maxigobot.WithFormat(maxigo.FormatMarkdown),          // markdown formatting
    maxigobot.WithAttachments(                            // inline keyboard
        maxigo.NewInlineKeyboardAttachment(buttons),
    ),
    maxigobot.WithDisableLinkPreview(),                   // no link preview
)
```

| Option | Description |
|--------|-------------|
| `WithReplyTo(msgID)` | Reply to a specific message |
| `WithNotify(bool)` | Enable/disable notification for chat members |
| `WithFormat(format)` | Text format: `maxigo.FormatMarkdown` or `maxigo.FormatHTML` |
| `WithAttachments(att...)` | Attach files, keyboards, locations, etc. |
| `WithDisableLinkPreview()` | Prevent server from generating link previews |

### Responding to Callbacks

```go
// Notification (small toast at the top)
c.Respond("Done!")

// Alert (dialog that must be dismissed)
c.RespondAlert("Are you sure?")
```

### Key-Value Store

Context provides a thread-safe key-value store for passing data between middleware and handlers:

```go
// In middleware
func AuthMiddleware() maxigobot.MiddlewareFunc {
    return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
        return func(c maxigobot.Context) error {
            c.Set("user_role", "admin")
            return next(c)
        }
    }
}

// In handler
b.Handle("/dashboard", func(c maxigobot.Context) error {
    role := c.Get("user_role").(string)
    return c.Send("Your role: " + role)
})
```

### Direct API Access

For operations not covered by Context, access the underlying maxigo-client:

```go
b.Handle("/members", func(c maxigobot.Context) error {
    members, err := c.API().GetMembers(c.Ctx(), c.Chat(), maxigo.GetMembersOpts{Count: 100})
    if err != nil {
        return err
    }
    return c.Send(fmt.Sprintf("Chat has %d members", len(members.Members)))
})
```

## Error Handling

### Global Error Handler

```go
b.OnError = func(err error, c maxigobot.Context) {
    log.Printf("Error: %v", err)
    if c != nil {
        c.Send("Something went wrong. Please try again.")
    }
}
```

If `OnError` is nil, errors are logged to stderr.

### BotError

Context methods return `*BotError` when an operation cannot be performed:

```go
var botErr *maxigobot.BotError
if errors.As(err, &botErr) {
    log.Printf("Endpoint: %s, Err: %v", botErr.Endpoint, botErr.Err)
}
```

### Sentinel Errors

| Error | Cause |
|-------|-------|
| `ErrNoChatID` | Update has no chat ID (e.g., trying to `Send` from an event without chat) |
| `ErrNoMessage` | Update has no message (e.g., trying to `Edit` from a lifecycle hook) |
| `ErrNoCallback` | Update is not a callback (e.g., trying to `Respond` from a text message) |
| `ErrNilPhoto` | `SendPhoto` called with nil payload |
| `ErrAlreadyStarted` | `Start()` called more than once |

### Panic Recovery

The bot recovers from panics in handlers and in the poller. Recovered panics are passed to `OnError` (or logged if nil).

## Lifecycle

```go
b, _ := maxigobot.New("TOKEN")

// Register handlers and middleware...

// Start blocks until Stop() is called.
go b.Start()

// Graceful shutdown.
// Signals the poller to stop, waits for in-flight handlers to complete.
b.Stop() // safe to call multiple times
```

`Start()` panics if called more than once.

## Testing

Use `WithClient` to inject a mock maxigo-client:

```go
func TestBot(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock API responses
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(maxigo.SimpleQueryResult{Success: true})
    }))
    defer srv.Close()

    client, _ := maxigo.New("test-token", maxigo.WithBaseURL(srv.URL))
    b, _ := maxigobot.New("test-token", maxigobot.WithClient(client))

    // Register handlers, then test by calling processUpdate directly
    // or by sending updates through the poller channel.
}
```

## Ecosystem

| Package | Description |
|---------|-------------|
| [maxigo-client](https://github.com/maxigo-bot/maxigo-client) | Idiomatic Go HTTP client for Max Bot API (zero external deps) |
| [maxigo-bot](https://github.com/maxigo-bot/maxigo-bot) | Gin-style bot framework with router/middleware/context |

## License

MIT
