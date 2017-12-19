package chain

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Middleware is a function that takes a HTTP handler
// and returns a modified http handler
type Middleware func(http.HandlerFunc) http.HandlerFunc

type Chain struct {
	route       *mux.Route
	middlewares []Middleware
}

func nullMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return h
}

// New constructs an empty Chain
func New() Chain {
	return Chain{route: &mux.Route{}, middlewares: []Middleware{nullMiddleware}}
}

// FromRoute constructs an empty Chain from a mux Route
func FromRoute(r *mux.Route) Chain {
	return Chain{route: r, middlewares: []Middleware{nullMiddleware}}
}

// Add adds a middleware to a Chain
func (c Chain) Add(m Middleware) Chain {
	return Chain{middlewares: append(c.middlewares, m), route: c.route}
}

// ToRoute converts the Chain to a normal HTTP handler and binds it to the route
func (c Chain) ToRoute(routeHandler http.HandlerFunc) {
	c.route.HandlerFunc(c.ToMiddleware()(routeHandler))
}

// ToMiddleware returns the middleware of a Chain
func (c Chain) ToMiddleware() Middleware {
	return func(h http.HandlerFunc) http.HandlerFunc {
		for i := len(c.middlewares) - 1; i >= 0; i-- {
			m := c.middlewares[i]
			h = m(h)
		}
		return h
	}
}
