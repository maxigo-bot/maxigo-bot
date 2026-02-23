# Changelog

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
