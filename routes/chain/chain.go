package chain

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Handler is like http.HandlerFunc, but returns an error indicating a failure
// during process the request. This can be used for cases when the application
// wishes to serve a 500 Internal Server Error.
type Handler func(http.ResponseWriter, *http.Request) error

// Middleware is a function that takes a Handler and returns a Handler
// It describes a transformation on the request/response
// Middleware can short-circuit the chain by not calling the passed in handler.
type Middleware func(Handler) Handler

// TerminatingMiddleware should sit at the top of the middleware chain, and
// converts a Handler (which returns an error) into a standard http.HandlerFunc
// which can be given to things like mux.Router.
type TerminatingMiddleware func(Handler) http.HandlerFunc

// Chain represents a "chain" of middlewares through which a request can be
// threaded. Each middleware can modify the request/response and is responsible
// for calling the next middleware in the chain. The errorHandler is the last
// middleware in the chain, and converts it from a chain.Handler to a
// http.HandlerFunc.
type Chain struct {
	route        *mux.Route
	middlewares  []Middleware
	errorHandler TerminatingMiddleware
}

func nullMiddleware(h Handler) Handler {
	return h
}

// New constructs an empty Chain
func New(errorHandler TerminatingMiddleware) Chain {
	if errorHandler == nil {
		log.Panicf("Cannot create chain without errorHandler")
	}

	return Chain{
		route:        &mux.Route{},
		middlewares:  []Middleware{nullMiddleware},
		errorHandler: errorHandler,
	}
}

// Route adds a mux route to the Chain
func (c Chain) Route(r *mux.Route) Chain {
	return Chain{
		route:        r,
		middlewares:  c.middlewares,
		errorHandler: c.errorHandler,
	}
}

// Add adds a middleware to a Chain
func (c Chain) Add(m Middleware) Chain {
	return Chain{middlewares: append(c.middlewares, m), route: c.route, errorHandler: c.errorHandler}
}

// Resolve converts the Chain to a normal HTTP handler and binds it to the route
func (c Chain) Resolve(routeHandler Handler) {
	c.route.HandlerFunc(
		c.errorHandler(c.ToMiddleware()(routeHandler)),
	)
}

// ToMiddleware returns the middleware of a Chain
func (c Chain) ToMiddleware() Middleware {
	return func(h Handler) Handler {
		for i := len(c.middlewares) - 1; i >= 0; i-- {
			m := c.middlewares[i]
			h = m(h)
		}
		return h
	}
}
