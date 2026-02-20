package maxigobot

import "fmt"

// BotError represents an error that occurred while processing an update.
type BotError struct {
	// Endpoint is the handler endpoint where the error occurred.
	Endpoint string
	// Err is the underlying error.
	Err error
}

func (e *BotError) Error() string {
	if e.Endpoint != "" {
		return fmt.Sprintf("maxigobot: handler %q: %v", e.Endpoint, e.Err)
	}
	return fmt.Sprintf("maxigobot: %v", e.Err)
}

func (e *BotError) Unwrap() error {
	return e.Err
}
