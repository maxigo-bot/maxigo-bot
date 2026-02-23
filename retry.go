package maxigobot

import (
	gocontext "context"
	"errors"
	"net/http"
	"strings"
	"time"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

// DefaultRateLimitIntervals defines the default retry schedule for HTTP 429 errors.
var DefaultRateLimitIntervals = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
}

// DefaultUploadRetryIntervals defines the default retry schedule for
// "file not processed" errors (HTTP 400 with "not.processed" message).
var DefaultUploadRetryIntervals = []time.Duration{
	200 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
}

// retryConfig holds retry intervals for different error types.
type retryConfig struct {
	rateLimitIntervals    []time.Duration
	uploadRetryIntervals  []time.Duration
}

// intervalsFor returns the retry intervals appropriate for the given error.
// Returns nil if the error is not retryable.
func intervalsFor(err error, cfg retryConfig) []time.Duration {
	var e *maxigo.Error
	if !errors.As(err, &e) || e.Kind != maxigo.ErrAPI {
		return nil
	}
	if e.StatusCode == http.StatusTooManyRequests {
		return cfg.rateLimitIntervals
	}
	if e.StatusCode == http.StatusBadRequest && strings.Contains(e.Message, "not.processed") {
		return cfg.uploadRetryIntervals
	}
	return nil
}

// withRetry executes fn and retries on retryable errors using intervals
// determined by the error type. Returns nil on success or the last error
// if all attempts are exhausted. Respects context cancellation between retries.
func withRetry(ctx gocontext.Context, cfg retryConfig, fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}

	intervals := intervalsFor(err, cfg)
	for _, d := range intervals {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}

		err = fn()
		if err == nil {
			return nil
		}
		if intervalsFor(err, cfg) == nil {
			return err
		}
	}

	return err
}
