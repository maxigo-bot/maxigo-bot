package middleware

import (
	"log"
	"time"

	maxigobot "github.com/maxigo-bot/maxigo-bot"
)

// LogFunc defines a function for logging. Compatible with log.Printf, fmt.Printf, etc.
type LogFunc func(format string, args ...any)

// LoggerConfig defines the config for Logger middleware.
type LoggerConfig struct {
	// Skipper defines a function to skip this middleware.
	Skipper Skipper

	// Log defines the logging function. Default: log.Printf.
	Log LogFunc
}

// DefaultLoggerConfig is the default Logger middleware config.
var DefaultLoggerConfig = LoggerConfig{
	Skipper: DefaultSkipper,
	Log:     log.Printf,
}

// Logger returns a Logger middleware with default config.
func Logger() maxigobot.MiddlewareFunc {
	return LoggerWithConfig(DefaultLoggerConfig)
}

// LoggerWithConfig returns a Logger middleware with custom config.
func LoggerWithConfig(cfg LoggerConfig) maxigobot.MiddlewareFunc {
	if cfg.Skipper == nil {
		cfg.Skipper = DefaultLoggerConfig.Skipper
	}
	if cfg.Log == nil {
		cfg.Log = DefaultLoggerConfig.Log
	}

	return func(next maxigobot.HandlerFunc) maxigobot.HandlerFunc {
		return func(c maxigobot.Context) error {
			if cfg.Skipper(c) {
				return next(c)
			}

			start := time.Now()
			err := next(c)
			duration := time.Since(start)

			updateType := ""
			if u := c.Update(); u.UpdateType != "" {
				updateType = string(u.UpdateType)
			}

			senderID := int64(0)
			if s := c.Sender(); s != nil {
				senderID = s.UserID
			}

			chatID := c.Chat()

			if err != nil {
				cfg.Log("%-20s | sender=%-10d | chat=%-10d | %s | error: %v",
					updateType, senderID, chatID, duration, err)
			} else {
				cfg.Log("%-20s | sender=%-10d | chat=%-10d | %s",
					updateType, senderID, chatID, duration)
			}

			return err
		}
	}
}
