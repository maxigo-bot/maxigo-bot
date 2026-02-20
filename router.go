package maxigobot

import (
	"strings"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

// parseCommand extracts command name and payload from text.
// Format: /command:payload → ("command", "payload", true)
// Non-command text returns ("", "", false).
func parseCommand(text string) (command, payload string, isCommand bool) {
	if text == "" || text[0] != '/' {
		return "", "", false
	}

	// Remove leading "/".
	text = text[1:]

	if idx := strings.IndexByte(text, ':'); idx >= 0 {
		return text[:idx], text[idx+1:], true
	}

	return text, "", true
}

// resolveEndpoint determines the endpoint key for a given update.
// Returns the endpoint string and the concrete update pointer.
func resolveEndpoint(raw any) (endpoint string, command string, payload string) {
	switch u := raw.(type) {
	case *maxigo.MessageCreatedUpdate:
		if u.Message.Body.Text != nil {
			cmd, pl, isCmd := parseCommand(*u.Message.Body.Text)
			if isCmd {
				return "/" + cmd, cmd, pl
			}
			// Has text but not a command.
			return OnText, "", ""
		}
		// No text (photo, sticker, etc.) — route to OnMessage.
		return OnMessage, "", ""

	case *maxigo.MessageCallbackUpdate:
		return OnCallback(u.Callback.Payload), "", ""

	case *maxigo.MessageEditedUpdate:
		return OnEdited, "", ""
	case *maxigo.MessageRemovedUpdate:
		return OnRemoved, "", ""

	// Lifecycle hooks.
	case *maxigo.BotStartedUpdate:
		return OnBotStarted, "", ""
	case *maxigo.BotStoppedUpdate:
		return OnBotStopped, "", ""
	case *maxigo.BotAddedUpdate:
		return OnBotAdded, "", ""
	case *maxigo.BotRemovedUpdate:
		return OnBotRemoved, "", ""
	case *maxigo.UserAddedUpdate:
		return OnUserAdded, "", ""
	case *maxigo.UserRemovedUpdate:
		return OnUserRemoved, "", ""
	case *maxigo.ChatTitleChangedUpdate:
		return OnChatTitleChanged, "", ""
	case *maxigo.MessageChatCreatedUpdate:
		return OnChatCreated, "", ""
	case *maxigo.DialogMutedUpdate:
		return OnDialogMuted, "", ""
	case *maxigo.DialogUnmutedUpdate:
		return OnDialogUnmuted, "", ""
	case *maxigo.DialogClearedUpdate:
		return OnDialogCleared, "", ""
	case *maxigo.DialogRemovedUpdate:
		return OnDialogRemoved, "", ""

	default:
		return "", "", ""
	}
}

// findHandler looks up a handler in bot and group registries.
// For message_created, it tries: exact command → OnText → OnMessage.
func (b *Bot) findHandler(endpoint string, update any) (*handlerEntry, []MiddlewareFunc) {
	// Try exact match in groups first, then bot handlers.
	if entry, groupMW := b.findInGroups(endpoint); entry != nil {
		return entry, groupMW
	}
	if entry, ok := b.handlers[endpoint]; ok {
		return entry, nil
	}

	// Fallback chain for message_created commands.
	if _, ok := update.(*maxigo.MessageCreatedUpdate); ok {
		if endpoint != OnText && endpoint != OnMessage {
			// Command not found — try OnText.
			if entry, groupMW := b.findInGroups(OnText); entry != nil {
				return entry, groupMW
			}
			if entry, ok := b.handlers[OnText]; ok {
				return entry, nil
			}
		}
		if endpoint != OnMessage {
			// Try OnMessage as catch-all.
			if entry, groupMW := b.findInGroups(OnMessage); entry != nil {
				return entry, groupMW
			}
			if entry, ok := b.handlers[OnMessage]; ok {
				return entry, nil
			}
		}
	}

	// Fallback for callbacks: try catch-all callback handler.
	if _, ok := update.(*maxigo.MessageCallbackUpdate); ok && endpoint != OnCallback("") {
		if entry, groupMW := b.findInGroups(OnCallback("")); entry != nil {
			return entry, groupMW
		}
		if entry, ok := b.handlers[OnCallback("")]; ok {
			return entry, nil
		}
	}

	return nil, nil
}

func (b *Bot) findInGroups(endpoint string) (*handlerEntry, []MiddlewareFunc) {
	for _, g := range b.groups {
		if entry, ok := g.handlers[endpoint]; ok {
			return entry, g.middleware
		}
	}
	return nil, nil
}
