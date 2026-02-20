# maxigo-bot

[![Go Reference](https://pkg.go.dev/badge/github.com/maxigo-bot/maxigo-bot.svg)](https://pkg.go.dev/github.com/maxigo-bot/maxigo-bot)
[![Go Report Card](https://goreportcard.com/badge/github.com/maxigo-bot/maxigo-bot)](https://goreportcard.com/report/github.com/maxigo-bot/maxigo-bot)
[![CI](https://github.com/maxigo-bot/maxigo-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/maxigo-bot/maxigo-bot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/maxigo-bot/maxigo-bot/branch/main/graph/badge.svg)](https://codecov.io/gh/maxigo-bot/maxigo-bot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/maxigo-bot/maxigo-bot)](https://github.com/maxigo-bot/maxigo-bot)

Bot framework for [Max messenger](https://max.ru). Router, middleware, context, groups — inspired by [Echo](https://echo.labstack.com) and [telebot](https://github.com/tucnak/telebot).

## Documentation

- **[English Guide](docs/guide.md)** — full framework reference with examples
- **[Документация на русском](docs/guide-ru.md)** — полное описание фреймворка с примерами

## Installation

```bash
go get github.com/maxigo-bot/maxigo-bot
```

Requires Go 1.25+.

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

	b.Start()
}
```

## Features

- Gin/Echo-style routing — commands, events, callbacks
- Two-level middleware: `Pre` (all updates) and `Use` (matched only)
- Handler groups with isolated middleware stacks
- Rich `Context` — send, reply, edit, delete, respond to callbacks
- Long polling with exponential backoff and graceful shutdown
- Built on [maxigo-client](https://github.com/maxigo-bot/maxigo-client) — zero external transitive dependencies
- Full Max Bot API update coverage (16 update types)

## Routing

### Commands

Max uses `:` as command separator (not space like Telegram): `/start:payload`.

```go
b.Handle("/start", func(c maxigobot.Context) error {
    log.Printf("Command: %s, Payload: %s", c.Command(), c.Payload())
    return c.Send("Welcome!")
})

b.Handle("/help", helpHandler)
```

### Events

```go
b.Handle(maxigobot.OnText, textHandler)           // text messages (not commands)
b.Handle(maxigobot.OnMessage, catchAllHandler)     // any message (fallback)
b.Handle(maxigobot.OnBotStarted, startHandler)     // user pressed Start
b.Handle(maxigobot.OnEdited, editedHandler)         // message edited
```

### Callbacks

```go
// Exact payload match.
b.Handle(maxigobot.OnCallback("confirm"), func(c maxigobot.Context) error {
    return c.Respond("Confirmed!")
})

// Catch-all callback handler.
b.Handle(maxigobot.OnCallback(""), func(c maxigobot.Context) error {
    log.Printf("Unknown callback: %s", c.Data())
    return nil
})
```

### Fallback Chain

For `message_created` updates, routing tries: **exact command** → `OnText` → `OnMessage`.
For callbacks: **exact payload** → `OnCallback("")`.

## Middleware

```go
// Pre-middleware — runs on ALL updates, before routing.
b.Pre(recoverMiddleware)

// Use-middleware — runs only on matched handlers, after routing.
b.Use(loggerMiddleware)
```

**Execution order:**

```
Update → Pre-middleware → Routing → Use-middleware → Group middleware → Per-handler middleware → Handler
```

**Writing middleware:**

```go
func Logger() maxigobot.MiddlewareFunc {
    return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
        return func(c maxigobot.Context) error {
            log.Printf("[%d] %s", c.Chat(), c.Text())
            return next(c)
        }
    }
}
```

## Groups

Groups provide isolated middleware stacks for a subset of handlers:

```go
admin := b.Group()
admin.Use(adminOnlyMiddleware)
admin.Handle("/ban", banHandler)
admin.Handle("/mute", muteHandler)

// Regular handlers are unaffected.
b.Handle("/help", helpHandler)
```

## Context

`Context` provides access to the current update and bot API:

```go
b.Handle("/info", func(c maxigobot.Context) error {
    // Update data
    sender := c.Sender()     // *maxigo.User
    chatID := c.Chat()       // int64
    msg := c.Message()       // *maxigo.Message
    text := c.Text()         // full message text
    cmd := c.Command()       // "info" (without "/")
    payload := c.Payload()   // text after ":"
    args := c.Args()         // payload split by whitespace

    // Sending
    c.Send("text")                                // send to chat
    c.Reply("text")                               // reply to message
    c.Edit("new text")                            // edit current message
    c.Delete()                                    // delete current message

    // Callbacks
    c.Respond("notification")                     // answer callback
    c.Data()                                      // callback payload

    // Typing indicator
    c.Notify(maxigo.ActionTyping)

    // Direct API access
    c.API().GetChat(c.Ctx(), chatID)

    // Key-value store (thread-safe)
    c.Set("user_role", "admin")
    role := c.Get("user_role")
})
```

### Send Options

```go
c.Send("hello",
    maxigobot.WithReplyTo(messageID),
    maxigobot.WithNotify(false),
    maxigobot.WithFormat(maxigo.FormatMarkdown),
    maxigobot.WithAttachments(maxigo.NewInlineKeyboardAttachment(buttons)),
    maxigobot.WithDisableLinkPreview(),
)
```

## Event Constants

| Constant             | Update Type            | Description                 |
|----------------------|------------------------|-----------------------------|
| `OnText`             | `message_created`      | Text message (not command)  |
| `OnMessage`          | `message_created`      | Any message (catch-all)     |
| `OnEdited`           | `message_edited`       | Message edited              |
| `OnRemoved`          | `message_removed`      | Message removed             |
| `OnBotStarted`       | `bot_started`          | User pressed Start          |
| `OnBotStopped`       | `bot_stopped`          | User stopped/blocked bot    |
| `OnBotAdded`         | `bot_added`            | Bot added to chat           |
| `OnBotRemoved`       | `bot_removed`          | Bot removed from chat       |
| `OnUserAdded`        | `user_added`           | User added to chat          |
| `OnUserRemoved`      | `user_removed`         | User removed from chat      |
| `OnChatTitleChanged`  | `chat_title_changed`   | Chat title changed          |
| `OnChatCreated`      | `message_chat_created` | Chat created via button     |
| `OnDialogMuted`      | `dialog_muted`         | User muted dialog           |
| `OnDialogUnmuted`    | `dialog_unmuted`       | User unmuted dialog         |
| `OnDialogCleared`    | `dialog_cleared`       | User cleared dialog history |
| `OnDialogRemoved`    | `dialog_removed`       | User removed dialog         |
| `OnCallback("id")`   | `message_callback`     | Callback with payload       |

## Options

```go
b, err := maxigobot.New("TOKEN",
    maxigobot.WithLongPolling(30),                // long polling with 30s timeout
    maxigobot.WithClient(preConfiguredClient),     // inject maxigo-client
    maxigobot.WithUpdateTypes("message_created",   // filter update types
        "message_callback"),
)
```

## Error Handling

```go
// Global error handler.
b.OnError = func(err error, c maxigobot.Context) {
    log.Printf("Error: %v", err)
    if c != nil {
        c.Send("Something went wrong.")
    }
}
```

Bot errors can be unwrapped:

```go
var botErr *maxigobot.BotError
if errors.As(err, &botErr) {
    log.Printf("Endpoint: %s, Err: %v", botErr.Endpoint, botErr.Err)
}
```

## Ecosystem

| Package | Description |
|---------|-------------|
| [maxigo-client](https://github.com/maxigo-bot/maxigo-client) | Idiomatic Go HTTP client for Max Bot API (zero external deps) |
| [maxigo-bot](https://github.com/maxigo-bot/maxigo-bot) | Bot framework with router, middleware, and context |

## License

MIT
