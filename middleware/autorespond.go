package middleware

import maxigobot "github.com/maxigo-bot/maxigo-bot"

// AutoRespondConfig defines the config for AutoRespond middleware.
type AutoRespondConfig struct {
	// Skipper defines a function to skip this middleware.
	Skipper Skipper
}

// DefaultAutoRespondConfig is the default AutoRespond middleware config.
var DefaultAutoRespondConfig = AutoRespondConfig{
	Skipper: DefaultSkipper,
}

// AutoRespond returns a middleware that automatically answers callback queries
// after the handler completes. This removes the loading state from callback buttons.
func AutoRespond() maxigobot.MiddlewareFunc {
	return AutoRespondWithConfig(DefaultAutoRespondConfig)
}

// AutoRespondWithConfig returns an AutoRespond middleware with custom config.
func AutoRespondWithConfig(cfg AutoRespondConfig) maxigobot.MiddlewareFunc {
	if cfg.Skipper == nil {
		cfg.Skipper = DefaultAutoRespondConfig.Skipper
	}

	return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
		return func(c maxigobot.Context) error {
			if cfg.Skipper(c) {
				return next(c)
			}

			err := next(c)

			if c.Callback() != nil {
				// Ignore respond errors — the handler may have already responded.
				_ = c.Respond("")
			}

			return err
		}
	}
}
