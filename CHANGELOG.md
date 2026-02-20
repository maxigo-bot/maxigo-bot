# Changelog

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
