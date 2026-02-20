package maxigobot

import "fmt"

// Group represents a handler group with an isolated middleware stack.
// Handlers registered in a group inherit the group's middleware
// in addition to global Use-middleware.
type Group struct {
	bot        *Bot
	middleware []MiddlewareFunc
	handlers   map[string]*handlerEntry
}

// Use appends middleware to the group's middleware stack.
func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

// Handle registers a handler for the given endpoint within this group.
func (g *Group) Handle(endpoint any, h HandlerFunc, m ...MiddlewareFunc) {
	key := endpointKey(endpoint)
	g.handlers[key] = &handlerEntry{
		handler:    h,
		middleware: m,
	}
}

// endpointKey converts an endpoint to its map key.
// Panics if endpoint is not a string, since Handle is called at setup time.
func endpointKey(endpoint any) string {
	switch e := endpoint.(type) {
	case string:
		return e
	default:
		panic(fmt.Sprintf("maxigobot: unsupported endpoint type %T", endpoint))
	}
}
