package middleware

import maxigobot "github.com/maxigo-bot/maxigo-bot"

// StripBotMention returns a middleware that removes leading "@..." mention
// tokens from the message text in group chats, so that c.Text() in handlers
// returns the clean text without the bot tag.
//
// The middleware mutates the message body in place. It is intended to be
// installed as a Pre-middleware: b.Pre(middleware.StripBotMention()).
//
// In dialogs the text is left untouched. Built-in command routing already
// strips mentions before parsing commands, so install this middleware only
// when you want OnText/OnMessage handlers to see the cleaned text.
func StripBotMention() maxigobot.MiddlewareFunc {
	return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
		return func(c maxigobot.Context) error {
			msg := c.Message()
			if msg != nil && msg.Body.Text != nil {
				if cleaned, stripped := maxigobot.StripBotMention(*msg.Body.Text, msg.Recipient.ChatType); stripped {
					*msg.Body.Text = cleaned
				}
			}
			return next(c)
		}
	}
}
