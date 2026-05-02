# Changelog

## [v0.4.0] - 2026-05-02

### Fixed
- Commands prefixed with a bot mention in group chats are now routed correctly. Messages like `@bot_id /start` and `@bot_id /start:payload` were previously falling through to `OnText` because the leading `@` prevented command parsing. The router now strips leading `@...` mention tokens in group chats before parsing the command. Multiple consecutive mentions are stripped (e.g. `@someone @bot_id /start`). Mention-only messages (`@bot_id`) are dropped.

### Added
- `maxigobot.StripBotMention(text, chatType)` — public helper that removes leading `@...` mention tokens from group-chat messages.
- `middleware.StripBotMention()` — opt-in middleware that mutates the message text in place so that `c.Text()` in `OnText`/`OnMessage` handlers returns the cleaned text. Built-in command routing works without this middleware; install it only when handlers need the clean text.

### Changed
- Bumped `github.com/maxigo-bot/maxigo-client` from `v0.3.0` to `v0.5.0`. Client adds `WithRetry()` option, `CheckPhoneNumbers`, `SendMessageToPhones`, upload convenience methods (`UploadPhotoFromFile`, `UploadPhotoFromURL`, `UploadMediaFromFile`, `UploadMediaFromURL`), 5 new `ChatAdminPermission` constants, and SSRF/path-traversal hardening for URL uploads. No breaking changes.

### Credits
- Original report and fix proposal: [@poluvasyan](https://github.com/poluvasyan) ([#1](https://github.com/maxigo-bot/maxigo-bot/pull/1)).

## [v0.3.1] - 2026-04-01

### Fixed
- Poller errors (poll failures, update parse errors, panics) now route through `OnError` instead of `log.Printf`
- Error wrapping uses `%w` to preserve error chain for `errors.Is`/`errors.As`

## [v0.3.0] - 2026-02-23

### Added
- **Automatic retry** with separate strategies for different transient API errors:
  - `WithRateLimitIntervals(...)` — retry schedule for HTTP 429 errors (disabled by default)
  - `WithUploadRetryIntervals(...)` — retry schedule for "file not processed" errors (enabled by default: 200ms, 500ms, 1s, 2s)
  - Context-aware: respects cancellation between retries
- **Attachment-based event routing**: `OnContact`, `OnPhoto`, `OnLocation` events for messages with attachments
  - Attachment events take priority over `OnText` in routing
  - Backward-compatible: if no handler is registered, falls back to `OnText` → `OnMessage`

## [v0.2.0] - 2026-02-21

### Added
- **Middleware package** (`middleware/`): built-in middleware with Echo-style Config pattern and Skipper support
  - `Recover` / `RecoverWithConfig` — catches panics in handlers, converts to errors with optional stack trace
  - `Logger` / `LoggerWithConfig` — logs update type, sender ID, chat ID, duration, and errors via customizable `LogFunc`
  - `Whitelist` / `Blacklist` — filters updates by user ID (nil sender handling included)
  - `AutoRespond` / `AutoRespondWithConfig` — automatically answers callback queries after handler completes
- **`WithKeyboard`** send option — shorthand for sending inline keyboard buttons

## [v0.1.0] - 2026-02-20

Initial public release.

### Added
- **Bot**: creation via `New(token, opts...)` with functional options (`WithLongPolling`, `WithClient`, `WithUpdateTypes`), `Start()`/`Stop()` with graceful shutdown
- **Long Polling**: automatic long polling with exponential backoff on errors (1s → 30s), update marker tracking
- **Router**: routing by commands (`/start`), events (`OnText`, `OnMessage`, `OnEdited`, `OnRemoved`), and callback buttons (`OnCallback("id")`)
- **Context**: handler interface — `Send`, `Reply`, `Edit`, `Delete`, `SendPhoto`, `Respond`, `RespondAlert`, `Notify`, key-value store (`Get`/`Set`)
- **Middleware**: two-level system — `Pre` (all updates, before routing) and `Use` (matched handlers only, after routing)
- **Groups**: isolated handler groups with their own middleware via `bot.Group()`
- **16 events**: `OnText`, `OnMessage`, `OnEdited`, `OnRemoved`, `OnBotStarted`, `OnBotStopped`, `OnBotAdded`, `OnBotRemoved`, `OnUserAdded`, `OnUserRemoved`, `OnChatTitleChanged`, `OnChatCreated`, `OnDialogMuted`, `OnDialogUnmuted`, `OnDialogCleared`, `OnDialogRemoved`
- **Send options**: `WithReplyTo`, `WithNotify`, `WithFormat`, `WithAttachments`, `WithDisableLinkPreview`
- **Error handling**: `BotError` with endpoint and wrapped error
