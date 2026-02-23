package maxigobot

import (
	"encoding/json"
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

// attachmentTypeToEndpoint maps raw JSON attachment types to event endpoints.
var attachmentTypeToEndpoint = map[string]string{
	"contact":  OnContact,
	"image":    OnPhoto,
	"location": OnLocation,
}

// firstAttachmentType extracts the "type" field from the first raw attachment
// without fully parsing the attachment. Returns empty string if no attachments
// or the first attachment cannot be unmarshaled.
func firstAttachmentType(attachments []json.RawMessage) string {
	if len(attachments) == 0 {
		return ""
	}
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(attachments[0], &header); err != nil {
		return ""
	}
	return header.Type
}

// isAttachmentEndpoint reports whether the endpoint is an attachment-based event.
func isAttachmentEndpoint(ep string) bool {
	return ep == OnContact || ep == OnPhoto || ep == OnLocation
}

// resolveEndpoint determines the endpoint key for a given update.
// Returns the endpoint string and the concrete update pointer.
func resolveEndpoint(raw any) (endpoint string, command string, payload string) {
	switch u := raw.(type) {
	case *maxigo.MessageCreatedUpdate:
		// Check attachments first — attachment events take priority.
		if ep, ok := attachmentTypeToEndpoint[firstAttachmentType(u.Message.Body.Attachments)]; ok {
			return ep, "", ""
		}

		if u.Message.Body.Text != nil {
			cmd, pl, isCmd := parseCommand(*u.Message.Body.Text)
			if isCmd {
				return "/" + cmd, cmd, pl
			}
			// Has text but not a command.
			return OnText, "", ""
		}
		// No text, no recognized attachment — route to OnMessage.
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
// For message_created, it tries:
//   - Attachment endpoints: exact match → OnText (if message has text) → OnMessage
//   - Commands: exact match → OnText → OnMessage
func (b *Bot) findHandler(endpoint string, update any) (*handlerEntry, []MiddlewareFunc) {
	// Try exact match in groups first, then bot handlers.
	if entry, groupMW := b.findInGroups(endpoint); entry != nil {
		return entry, groupMW
	}
	if entry, ok := b.handlers[endpoint]; ok {
		return entry, nil
	}

	// Fallback chain for message_created.
	if u, ok := update.(*maxigo.MessageCreatedUpdate); ok {
		// Attachment endpoints fall back to OnText (if message has text) → OnMessage.
		if isAttachmentEndpoint(endpoint) {
			if u.Message.Body.Text != nil {
				if entry, groupMW := b.findInGroups(OnText); entry != nil {
					return entry, groupMW
				}
				if entry, ok := b.handlers[OnText]; ok {
					return entry, nil
				}
			}
			if entry, groupMW := b.findInGroups(OnMessage); entry != nil {
				return entry, groupMW
			}
			if entry, ok := b.handlers[OnMessage]; ok {
				return entry, nil
			}
			return nil, nil
		}

		// Commands fall back to OnText → OnMessage.
		if endpoint != OnText && endpoint != OnMessage {
			if entry, groupMW := b.findInGroups(OnText); entry != nil {
				return entry, groupMW
			}
			if entry, ok := b.handlers[OnText]; ok {
				return entry, nil
			}
		}
		if endpoint != OnMessage {
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
