// Package middleware provides built-in middleware for maxigo-bot.
package middleware

import maxigobot "github.com/maxigo-bot/maxigo-bot"

// Skipper defines a function to skip middleware.
// Returning true skips the middleware and calls the next handler directly.
type Skipper func(c maxigobot.Context) bool

// DefaultSkipper never skips — all updates pass through the middleware.
func DefaultSkipper(_ maxigobot.Context) bool { return false }
