package middleware

import (
	"fmt"
	"runtime/debug"

	maxigobot "github.com/maxigo-bot/maxigo-bot"
)

// RecoverConfig defines the config for Recover middleware.
type RecoverConfig struct {
	// Skipper defines a function to skip this middleware.
	Skipper Skipper

	// StackSize is the maximum size of the stack trace to capture (in bytes).
	// Default: 4 KB.
	StackSize int

	// PrintStack controls whether the stack trace is included in the error.
	// Default: true.
	PrintStack bool
}

// DefaultRecoverConfig is the default Recover middleware config.
var DefaultRecoverConfig = RecoverConfig{
	Skipper:    DefaultSkipper,
	StackSize:  4 << 10, // 4 KB
	PrintStack: true,
}

// Recover returns a Recover middleware with default config.
func Recover() maxigobot.MiddlewareFunc {
	return RecoverWithConfig(DefaultRecoverConfig)
}

// RecoverWithConfig returns a Recover middleware with custom config.
func RecoverWithConfig(cfg RecoverConfig) maxigobot.MiddlewareFunc {
	if cfg.Skipper == nil {
		cfg.Skipper = DefaultRecoverConfig.Skipper
	}
	if cfg.StackSize == 0 {
		cfg.StackSize = DefaultRecoverConfig.StackSize
	}

	return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
		return func(c maxigobot.Context) (err error) {
			if cfg.Skipper(c) {
				return next(c)
			}

			defer func() {
				if r := recover(); r != nil {
					if cfg.PrintStack {
						stack := debug.Stack()
						if len(stack) > cfg.StackSize {
							stack = stack[:cfg.StackSize]
						}
						err = fmt.Errorf("panic recovered: %v\n%s", r, stack)
					} else {
						err = fmt.Errorf("panic recovered: %v", r)
					}
				}
			}()

			return next(c)
		}
	}
}
