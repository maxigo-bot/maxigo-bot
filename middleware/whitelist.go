package middleware

import maxigobot "github.com/maxigo-bot/maxigo-bot"

// Whitelist returns a middleware that only allows updates from the given user IDs.
// If the sender is nil (e.g. MessageRemovedUpdate), the update is silently dropped.
func Whitelist(userIDs ...int64) maxigobot.MiddlewareFunc {
	allowed := make(map[int64]struct{}, len(userIDs))
	for _, id := range userIDs {
		allowed[id] = struct{}{}
	}

	return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
		return func(c maxigobot.Context) error {
			s := c.Sender()
			if s == nil {
				return nil
			}
			if _, ok := allowed[s.UserID]; !ok {
				return nil
			}
			return next(c)
		}
	}
}

// Blacklist returns a middleware that blocks updates from the given user IDs.
// If the sender is nil (e.g. MessageRemovedUpdate), the update passes through.
func Blacklist(userIDs ...int64) maxigobot.MiddlewareFunc {
	blocked := make(map[int64]struct{}, len(userIDs))
	for _, id := range userIDs {
		blocked[id] = struct{}{}
	}

	return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
		return func(c maxigobot.Context) error {
			s := c.Sender()
			if s == nil {
				return next(c)
			}
			if _, ok := blocked[s.UserID]; ok {
				return nil
			}
			return next(c)
		}
	}
}
