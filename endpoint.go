package maxigobot

// Event endpoints for message-related updates.
const (
	// OnText matches message_created updates containing text that is not a command.
	OnText = "\atext"
	// OnMessage matches any message_created update (catch-all fallback).
	OnMessage = "\amessage"
	// OnEdited matches message_edited updates.
	OnEdited = "\aedited"
	// OnRemoved matches message_removed updates.
	OnRemoved = "\aremoved"
)

// Attachment-based event endpoints.
// These take priority over OnText when the message contains a matching attachment.
// If no handler is registered for the attachment event, routing falls back to OnText → OnMessage.
const (
	// OnContact matches message_created updates with a contact attachment.
	OnContact = "\acontact"
	// OnPhoto matches message_created updates with an image attachment.
	OnPhoto = "\aphoto"
	// OnLocation matches message_created updates with a location attachment.
	OnLocation = "\alocation"
)

// Lifecycle hook endpoints.
const (
	// OnBotStarted fires when a user presses the Start button.
	OnBotStarted = "\abot_started"
	// OnBotStopped fires when a user stops/blocks the bot.
	OnBotStopped = "\abot_stopped"
	// OnBotAdded fires when the bot is added to a chat.
	OnBotAdded = "\abot_added"
	// OnBotRemoved fires when the bot is removed from a chat.
	OnBotRemoved = "\abot_removed"
	// OnUserAdded fires when a user is added to a chat.
	OnUserAdded = "\auser_added"
	// OnUserRemoved fires when a user is removed from a chat.
	OnUserRemoved = "\auser_removed"
	// OnChatTitleChanged fires when a chat title is changed.
	OnChatTitleChanged = "\atitle_changed"
	// OnChatCreated fires when a chat is created via a button.
	OnChatCreated = "\achat_created"
	// OnDialogMuted fires when a user mutes the dialog.
	OnDialogMuted = "\adialog_muted"
	// OnDialogUnmuted fires when a user unmutes the dialog.
	OnDialogUnmuted = "\adialog_unmuted"
	// OnDialogCleared fires when a user clears dialog history.
	OnDialogCleared = "\adialog_cleared"
	// OnDialogRemoved fires when a user removes the dialog.
	OnDialogRemoved = "\adialog_removed"
)

// callbackPrefix is used to build callback endpoint keys.
const callbackPrefix = "\f"

// OnCallback returns an endpoint key for a callback with the given unique payload.
// Use an empty string to match all callbacks.
func OnCallback(unique string) string {
	return callbackPrefix + unique
}
