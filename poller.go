package maxigobot

import (
	gocontext "context"
	"encoding/json"
	"log"
	"runtime/debug"
	"time"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
)

// Poller defines the interface for receiving updates.
// Implementations must close the updates channel before returning.
type Poller interface {
	// Poll starts receiving updates, sending them to the updates channel.
	// It must block until stop is closed, and must close updates before returning.
	Poll(b *Bot, updates chan<- any, stop chan struct{})
}

// LongPoller implements Poller using long polling via GetUpdates.
type LongPoller struct {
	// Timeout is the long-polling timeout in seconds (default 30).
	Timeout int
	// UpdateTypes filters which update types to receive. Empty means all.
	UpdateTypes []string
}

// Poll starts the long-polling loop.
// It closes the updates channel before returning.
func (p *LongPoller) Poll(b *Bot, updates chan<- any, stop chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("maxigobot: panic in poller: %v\n%s", r, debug.Stack())
		}
	}()
	defer close(updates)

	// Create a cancellable context tied to the stop signal
	// so that in-flight GetUpdates calls are interrupted on shutdown.
	ctx, cancel := gocontext.WithCancel(gocontext.Background())
	defer cancel()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-stop:
			cancel()
		case <-done:
		}
	}()

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	var marker int64
	backoff := initialBackoff

	for {
		select {
		case <-stop:
			return
		default:
		}

		opts := maxigo.GetUpdatesOpts{
			Timeout: timeout,
			Marker:  marker,
			Types:   p.UpdateTypes,
			Limit:   100,
		}

		list, err := b.client.GetUpdates(ctx, opts)
		if err != nil {
			if ctx.Err() != nil {
				return // Shutdown requested, exit gracefully.
			}
			log.Printf("maxigobot: poll error (retry in %v): %v", backoff, err)
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
			case <-stop:
				timer.Stop()
				return
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		backoff = initialBackoff // Reset on success.

		for _, raw := range list.Updates {
			upd, err := ParseUpdate(raw)
			if err != nil {
				log.Printf("maxigobot: parse update error: %v", err)
				continue
			}
			if upd == nil {
				continue // Unknown update type, skip.
			}
			updates <- upd
		}

		if list.Marker != nil {
			marker = *list.Marker
		}
	}
}

// updateHeader is used to peek at the update_type discriminator.
type updateHeader struct {
	UpdateType maxigo.UpdateType `json:"update_type"`
}

// ParseUpdate unmarshals a raw JSON update into a concrete typed struct.
func ParseUpdate(data json.RawMessage) (any, error) {
	var header updateHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}

	var target any
	switch header.UpdateType {
	case maxigo.UpdateMessageCreated:
		target = new(maxigo.MessageCreatedUpdate)
	case maxigo.UpdateMessageCallback:
		target = new(maxigo.MessageCallbackUpdate)
	case maxigo.UpdateMessageEdited:
		target = new(maxigo.MessageEditedUpdate)
	case maxigo.UpdateMessageRemoved:
		target = new(maxigo.MessageRemovedUpdate)
	case maxigo.UpdateBotStarted:
		target = new(maxigo.BotStartedUpdate)
	case maxigo.UpdateBotStopped:
		target = new(maxigo.BotStoppedUpdate)
	case maxigo.UpdateBotAdded:
		target = new(maxigo.BotAddedUpdate)
	case maxigo.UpdateBotRemoved:
		target = new(maxigo.BotRemovedUpdate)
	case maxigo.UpdateUserAdded:
		target = new(maxigo.UserAddedUpdate)
	case maxigo.UpdateUserRemoved:
		target = new(maxigo.UserRemovedUpdate)
	case maxigo.UpdateChatTitleChanged:
		target = new(maxigo.ChatTitleChangedUpdate)
	case maxigo.UpdateMessageChatCreated:
		target = new(maxigo.MessageChatCreatedUpdate)
	case maxigo.UpdateDialogMuted:
		target = new(maxigo.DialogMutedUpdate)
	case maxigo.UpdateDialogUnmuted:
		target = new(maxigo.DialogUnmutedUpdate)
	case maxigo.UpdateDialogCleared:
		target = new(maxigo.DialogClearedUpdate)
	case maxigo.UpdateDialogRemoved:
		target = new(maxigo.DialogRemovedUpdate)
	default:
		// Unknown update type — skip silently for forward compatibility.
		return nil, nil
	}

	if err := json.Unmarshal(data, target); err != nil {
		return nil, err
	}
	return target, nil
}
