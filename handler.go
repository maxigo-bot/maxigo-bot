package maxigobot

// HandlerFunc defines a handler function for processing updates.
type HandlerFunc func(c Context) error

// MiddlewareFunc defines a middleware that wraps a handler.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// applyMiddleware wraps a handler with middleware in order.
// Middleware is applied so that the first in the slice executes first (outermost).
func applyMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	// Apply in reverse so middleware[0] is outermost.
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

// handlerEntry stores a handler with its per-handler middleware.
type handlerEntry struct {
	handler    HandlerFunc
	middleware []MiddlewareFunc
}
