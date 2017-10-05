package chain

import (
	"net/http"

	"github.com/gorilla/mux"
)

type httpHandler func(http.ResponseWriter, *http.Request)

type Chain struct {
	route   *mux.Route
	handler httpHandler
}

func New() Chain {
	return Chain{route: &mux.Route{}, handler: func(w http.ResponseWriter, h *http.Request) {}}
}

func (c Chain) Add(h httpHandler) Chain {
	newHandler := func(w http.ResponseWriter, r *http.Request) {
		c.handler(w, r)
		h(w, r)
	}
	return Chain{handler: newHandler, route: c.route}
}

func (c Chain) Resolve() httpHandler {
	return c.handler
}

func FromRoute(r *mux.Route) Chain {
	return Chain{route: r, handler: func(w http.ResponseWriter, h *http.Request) {}}
}

func (c Chain) ToRoute() {
	c.route.HandlerFunc(c.handler)
}
