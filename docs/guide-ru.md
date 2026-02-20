# maxigo-bot

Фреймворк для ботов [Max мессенджера](https://max.ru). Роутер, middleware, контекст, группы — вдохновлён [Echo](https://echo.labstack.com) и [telebot](https://github.com/tucnak/telebot).

> **[English Guide](guide.md)** | **[README](../README.md)**

## Установка

```bash
go get github.com/maxigo-bot/maxigo-bot
```

Требуется Go 1.25+. Построен на [maxigo-client](https://github.com/maxigo-bot/maxigo-client) — без внешних транзитивных зависимостей.

## Быстрый старт

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
        return c.Send("Привет, " + c.Sender().FirstName + "!")
    })

    b.Handle(maxigobot.OnText, func(c maxigobot.Context) error {
        return c.Reply("Вы написали: " + c.Text())
    })

    b.Start() // блокирует до вызова Stop()
}
```

## Конфигурация

Бот настраивается через функциональные опции:

```go
b, err := maxigobot.New("TOKEN",
    maxigobot.WithLongPolling(30),              // long polling с таймаутом 30с (по умолчанию)
    maxigobot.WithClient(preConfiguredClient),   // инжектировать готовый maxigo-client
    maxigobot.WithUpdateTypes(                   // фильтр типов обновлений
        "message_created",
        "message_callback",
    ),
)
```

### Опции

| Опция | Описание |
|-------|----------|
| `WithLongPolling(timeout)` | Таймаут long polling в секундах (по умолчанию: 30) |
| `WithClient(client)` | Инжектировать готовый `*maxigo.Client` (полезно для тестов) |
| `WithUpdateTypes(types...)` | Фильтровать типы обновлений, которые получает поллер |

### Доступ к клиенту

Для прямых API-вызовов:

```go
client := b.Client() // *maxigo.Client
bot, err := client.GetBot(ctx)
```

## Роутинг

### Команды

Max использует `:` как разделитель в командах (не пробел как в Telegram): `/start:payload`.

```go
b.Handle("/start", func(c maxigobot.Context) error {
    name := c.Sender().FirstName
    payload := c.Payload() // текст после ":"
    args := c.Args()       // payload разбитый по пробелам
    return c.Send("Добро пожаловать, " + name + "!")
})

b.Handle("/help", func(c maxigobot.Context) error {
    return c.Send("Доступные команды: /start, /help")
})
```

### События

```go
// Текстовые сообщения (не команды)
b.Handle(maxigobot.OnText, func(c maxigobot.Context) error {
    return c.Reply("Вы написали: " + c.Text())
})

// Любое сообщение (catch-all для message_created)
b.Handle(maxigobot.OnMessage, func(c maxigobot.Context) error {
    return c.Send("Получено сообщение")
})

// Хуки жизненного цикла
b.Handle(maxigobot.OnBotStarted, func(c maxigobot.Context) error {
    return c.Send("Добро пожаловать! Используйте /help для списка команд.")
})

b.Handle(maxigobot.OnBotAdded, func(c maxigobot.Context) error {
    return c.Send("Спасибо, что добавили меня!")
})

b.Handle(maxigobot.OnEdited, func(c maxigobot.Context) error {
    log.Printf("Сообщение отредактировано: %s", c.Text())
    return nil
})
```

### Callback-кнопки

```go
// Отправляем сообщение с инлайн-клавиатурой
b.Handle("/menu", func(c maxigobot.Context) error {
    return c.Send("Выберите действие:",
        maxigobot.WithAttachments(
            maxigo.NewInlineKeyboardAttachment([][]maxigo.Button{
                {
                    maxigo.NewCallbackButtonWithIntent("Подтвердить", "confirm", maxigo.IntentPositive),
                    maxigo.NewCallbackButtonWithIntent("Отмена", "cancel", maxigo.IntentNegative),
                },
            }),
        ),
    )
})

// Обработка конкретного callback
b.Handle(maxigobot.OnCallback("confirm"), func(c maxigobot.Context) error {
    return c.Respond("Подтверждено!")
})

b.Handle(maxigobot.OnCallback("cancel"), func(c maxigobot.Context) error {
    return c.Respond("Отменено.")
})

// Catch-all обработчик (пустая строка — любой неподходящий callback)
b.Handle(maxigobot.OnCallback(""), func(c maxigobot.Context) error {
    log.Printf("Неизвестный callback: %s", c.Data())
    return c.Respond("Неизвестное действие")
})
```

### Цепочка fallback

Для обновлений `message_created` роутер ищет обработчики в таком порядке:

1. **Точная команда** (`/start`, `/help`, ...) — совпадает первой
2. **`OnText`** — fallback для текстовых сообщений (включая ненайденные команды)
3. **`OnMessage`** — catch-all для любых сообщений (фото, стикеры и т.д.)

Для callback:

1. **Точный payload** (`OnCallback("confirm")`) — совпадает первым
2. **`OnCallback("")`** — catch-all для неподходящих callback

### Константы событий

| Константа | Тип обновления | Описание |
|-----------|----------------|----------|
| `OnText` | `message_created` | Текстовое сообщение (не команда) |
| `OnMessage` | `message_created` | Любое сообщение (catch-all) |
| `OnEdited` | `message_edited` | Сообщение отредактировано |
| `OnRemoved` | `message_removed` | Сообщение удалено |
| `OnBotStarted` | `bot_started` | Пользователь нажал Start |
| `OnBotStopped` | `bot_stopped` | Пользователь остановил бота |
| `OnBotAdded` | `bot_added` | Бот добавлен в чат |
| `OnBotRemoved` | `bot_removed` | Бот удалён из чата |
| `OnUserAdded` | `user_added` | Пользователь добавлен в чат |
| `OnUserRemoved` | `user_removed` | Пользователь удалён из чата |
| `OnChatTitleChanged` | `chat_title_changed` | Название чата изменено |
| `OnChatCreated` | `message_chat_created` | Чат создан через кнопку |
| `OnDialogMuted` | `dialog_muted` | Диалог замьючен |
| `OnDialogUnmuted` | `dialog_unmuted` | Диалог размьючен |
| `OnDialogCleared` | `dialog_cleared` | История диалога очищена |
| `OnDialogRemoved` | `dialog_removed` | Диалог удалён |
| `OnCallback("id")` | `message_callback` | Callback с конкретным payload |

## Middleware

Middleware оборачивает обработчики для добавления сквозной логики (логирование, авторизация, recovery и т.д.).

### Сигнатура

```go
type HandlerFunc func(c Context) error
type MiddlewareFunc func(next HandlerFunc) HandlerFunc
```

### Два уровня

**Pre-middleware** — выполняется для ВСЕХ обновлений, до роутинга:

```go
b.Pre(recoverMiddleware)
b.Pre(requestIDMiddleware)
```

**Use-middleware** — выполняется только для совпавших обработчиков, после роутинга:

```go
b.Use(loggerMiddleware)
b.Use(authMiddleware)
```

### Порядок выполнения

```
Обновление → Pre-middleware → Роутинг → Use-middleware → Group middleware → Per-handler middleware → Обработчик
```

Если обработчик не найден, Use-middleware и далее не выполняются.

### Написание middleware

```go
// Logger — логирует каждый обработанный апдейт
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

// Recover — перехватывает паники в обработчиках
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

// Whitelist — разрешает только определённым пользователям
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
            return c.Send("Доступ запрещён.")
        }
    }
}
```

### Middleware на конкретный обработчик

Можно привязать middleware к конкретному обработчику:

```go
b.Handle("/admin", adminHandler, adminOnlyMiddleware, auditMiddleware)
```

Они выполняются после глобальных и групповых middleware.

## Группы

Группы предоставляют изолированные стеки middleware для подмножества обработчиков:

```go
// Группа для администраторов
admin := b.Group()
admin.Use(Whitelist(123456, 789012))
admin.Handle("/ban", banHandler)
admin.Handle("/mute", muteHandler)
admin.Handle("/stats", statsHandler)

// Публичные обработчики — без admin middleware
b.Handle("/start", startHandler)
b.Handle("/help", helpHandler)
```

Группы наследуют глобальные `Use`-middleware и добавляют свои поверх:

```
Обновление → Pre → Роутинг → глобальные Use → group middleware → per-handler middleware → Обработчик
```

## Контекст

`Context` предоставляет обработчику доступ к текущему обновлению и API бота. Новый контекст создаётся для каждого обновления.

### Данные обновления

```go
b.Handle("/info", func(c maxigobot.Context) error {
    // Пользователь, инициировавший обновление (nil для некоторых событий)
    sender := c.Sender() // *maxigo.User

    // ID чата, где произошло обновление (0, если недоступен)
    chatID := c.Chat() // int64

    // Исходное сообщение (nil для хуков жизненного цикла)
    msg := c.Message() // *maxigo.Message

    // Полный текст сообщения (пустая строка, если не текстовое)
    text := c.Text() // string

    // Сырое обновление из maxigo-client
    upd := c.Update() // maxigo.Update

    // context.Context с привязкой к запросу
    ctx := c.Ctx() // context.Context
})
```

### Команды и payload

```go
// Для /greet:Иван Петров
b.Handle("/greet", func(c maxigobot.Context) error {
    c.Command() // "greet"
    c.Payload() // "Иван Петров"
    c.Args()    // ["Иван", "Петров"]
    return c.Send("Привет, " + c.Payload() + "!")
})

// BotStartedUpdate тоже имеет payload
b.Handle(maxigobot.OnBotStarted, func(c maxigobot.Context) error {
    ref := c.Payload() // start payload (deep link)
    return c.Send("Добро пожаловать! Ref: " + ref)
})
```

### Callback

```go
b.Handle(maxigobot.OnCallback("action"), func(c maxigobot.Context) error {
    cb := c.Callback() // *maxigo.Callback
    data := c.Data()   // строка payload callback

    // Ответить уведомлением
    return c.Respond("Действие получено: " + data)
})
```

### Отправка сообщений

```go
// Отправить в текущий чат
c.Send("Привет!")

// Ответить на текущее сообщение
c.Reply("Понял!")

// Отредактировать текущее сообщение
c.Edit("Обновлённый текст")

// Удалить текущее сообщение
c.Delete()

// Отправить фото
c.SendPhoto(&maxigo.PhotoAttachmentRequestPayload{
    Photos: tokens.Photos,
})

// Отправить индикатор набора текста
c.Notify(maxigo.ActionTypingOn)
```

### Опции отправки

```go
c.Send("Привет!",
    maxigobot.WithReplyTo(messageID),                    // ответ на конкретное сообщение
    maxigobot.WithNotify(false),                          // отключить уведомление
    maxigobot.WithFormat(maxigo.FormatMarkdown),          // форматирование markdown
    maxigobot.WithAttachments(                            // инлайн-клавиатура
        maxigo.NewInlineKeyboardAttachment(buttons),
    ),
    maxigobot.WithDisableLinkPreview(),                   // без превью ссылок
)
```

| Опция | Описание |
|-------|----------|
| `WithReplyTo(msgID)` | Ответить на конкретное сообщение |
| `WithNotify(bool)` | Включить/отключить уведомление для участников чата |
| `WithFormat(format)` | Формат текста: `maxigo.FormatMarkdown` или `maxigo.FormatHTML` |
| `WithAttachments(att...)` | Прикрепить файлы, клавиатуры, локации и т.д. |
| `WithDisableLinkPreview()` | Отключить генерацию превью ссылок |

### Ответ на callback

```go
// Уведомление (небольшой toast сверху)
c.Respond("Готово!")

// Алерт (диалог, который нужно закрыть)
c.RespondAlert("Вы уверены?")
```

### Хранилище ключ-значение

Контекст предоставляет потокобезопасное хранилище для передачи данных между middleware и обработчиками:

```go
// В middleware
func AuthMiddleware() maxigobot.MiddlewareFunc {
    return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
        return func(c maxigobot.Context) error {
            c.Set("user_role", "admin")
            return next(c)
        }
    }
}

// В обработчике
b.Handle("/dashboard", func(c maxigobot.Context) error {
    role := c.Get("user_role").(string)
    return c.Send("Ваша роль: " + role)
})
```

### Прямой доступ к API

Для операций, не покрытых Context, используйте maxigo-client напрямую:

```go
b.Handle("/members", func(c maxigobot.Context) error {
    members, err := c.API().GetMembers(c.Ctx(), c.Chat(), maxigo.GetMembersOpts{Count: 100})
    if err != nil {
        return err
    }
    return c.Send(fmt.Sprintf("В чате %d участников", len(members.Members)))
})
```

## Обработка ошибок

### Глобальный обработчик ошибок

```go
b.OnError = func(err error, c maxigobot.Context) {
    log.Printf("Ошибка: %v", err)
    if c != nil {
        c.Send("Произошла ошибка. Попробуйте ещё раз.")
    }
}
```

Если `OnError` равен nil, ошибки логируются в stderr.

### BotError

Методы контекста возвращают `*BotError`, когда операция не может быть выполнена:

```go
var botErr *maxigobot.BotError
if errors.As(err, &botErr) {
    log.Printf("Endpoint: %s, Err: %v", botErr.Endpoint, botErr.Err)
}
```

### Sentinel-ошибки

| Ошибка | Причина |
|--------|---------|
| `ErrNoChatID` | В обновлении нет ID чата (попытка `Send` из события без чата) |
| `ErrNoMessage` | В обновлении нет сообщения (попытка `Edit` из хука жизненного цикла) |
| `ErrNoCallback` | Обновление не является callback (попытка `Respond` из текстового сообщения) |
| `ErrNilPhoto` | `SendPhoto` вызван с nil payload |
| `ErrAlreadyStarted` | `Start()` вызван более одного раза |

### Восстановление после паник

Бот автоматически восстанавливается после паник в обработчиках и в поллере. Перехваченные паники передаются в `OnError` (или логируются, если nil).

## Жизненный цикл

```go
b, _ := maxigobot.New("TOKEN")

// Регистрация обработчиков и middleware...

// Start блокирует до вызова Stop()
go b.Start()

// Graceful shutdown.
// Сигнализирует поллеру остановиться, ждёт завершения текущих обработчиков.
b.Stop() // безопасно вызывать несколько раз
```

`Start()` паникует, если вызван более одного раза.

## Тестирование

Используйте `WithClient` для инжекции мок-клиента:

```go
func TestBot(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Мок ответов API
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(maxigo.SimpleQueryResult{Success: true})
    }))
    defer srv.Close()

    client, _ := maxigo.New("test-token", maxigo.WithBaseURL(srv.URL))
    b, _ := maxigobot.New("test-token", maxigobot.WithClient(client))

    // Регистрация обработчиков, затем тестирование через processUpdate
    // или отправку обновлений через канал поллера.
}
```

## Экосистема

| Пакет | Описание |
|-------|----------|
| [maxigo-client](https://github.com/maxigo-bot/maxigo-client) | Идиоматичный Go HTTP-клиент для Max Bot API (без внешних зависимостей) |
| [maxigo-bot](https://github.com/maxigo-bot/maxigo-bot) | Фреймворк для ботов с роутером, middleware и контекстом |

## Лицензия

MIT
