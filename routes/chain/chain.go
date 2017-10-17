package chain

import (
	"net/http"

	"github.com/gorilla/mux"
)

type httpHandler func(http.ResponseWriter, *http.Request)

// httpMiddleware is a handler that returns a bool indicating if the request
// should be propagated to the next middleware in the chain
type httpMiddleware func(http.ResponseWriter, *http.Request) bool

type Chain struct {
	route      *mux.Route
	middleware httpMiddleware
}

// New constructs a Chain with the given middleware as the first link
func New(first httpMiddleware) Chain {
	return Chain{route: &mux.Route{}, middleware: first}
}

// Add adds a middleware to the end of a Chain
func (c Chain) Add(h httpMiddleware) Chain {
	newHandler := func(w http.ResponseWriter, r *http.Request) bool {
		propagate := c.middleware(w, r)
		if propagate {
			return h(w, r)
		}
		return propagate
	}
	return Chain{middleware: newHandler, route: c.route}
}

// FromRoute constructs an empty Chain from a mux Route
func FromRoute(r *mux.Route) Chain {
	return Chain{route: r, middleware: func(w http.ResponseWriter, h *http.Request) bool { return true }}
}

// ToRoute converts the Chain to a normal HTTP handler and binds it to the route
func (c Chain) ToRoute(routeHandler httpHandler) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if c.middleware(w, r) {
			routeHandler(w, r)
		}
	}
	c.route.HandlerFunc(handler)
}

// ToMiddleware returns the middleware of a Chain
func (c Chain) ToMiddleware() httpMiddleware {
	return c.middleware
}
