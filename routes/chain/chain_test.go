package chain

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func nullHandler(w http.ResponseWriter, r *http.Request) {}
func TestAddMiddleware(t *testing.T) {
	log := make([]int, 0)

	middleware := func(http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log = append(log, 1)
		}
	}

	New().Add(middleware).ToMiddleware()(nullHandler)(nil, nil)

	assert.Equal(t, []int{1}, log)
}

func TestAddMultipleMiddleware(t *testing.T) {
	log := make([]int, 0)

	m1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log = append(log, 1)
			next(w, r)
		}
	}
	m2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log = append(log, 2)
			next(w, r)
		}
	}

	New().Add(m1).Add(m2).ToMiddleware()(nullHandler)(nil, nil)

	assert.Equal(t, []int{1, 2}, log)
}

func TestToRoute(t *testing.T) {
	log := make([]int, 0)

	m1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log = append(log, 1)
			next(w, r)
		}
	}
	m2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log = append(log, 2)
			next(w, r)
		}
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		log = append(log, 3)
	}

	route := mux.NewRouter().NewRoute()
	FromRoute(route).Add(m1).Add(m2).ToRoute(handler)
	route.GetHandler().ServeHTTP(nil, nil)

	assert.Equal(t, []int{1, 2, 3}, log)
}
