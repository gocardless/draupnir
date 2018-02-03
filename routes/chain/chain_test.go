package chain

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func nullHandler(w http.ResponseWriter, r *http.Request) error { return nil }

func testErrorHandler(t *testing.T) TerminatingMiddleware {
	return func(next Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := next(w, r)
			if err != nil {
				t.Fatalf("route raised error %s", err)
			}
		}
	}
}

func TestAddMiddleware(t *testing.T) {
	log := make([]int, 0)

	middleware := func(Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			log = append(log, 1)
			return nil
		}
	}

	New(testErrorHandler(t)).Add(middleware).ToMiddleware()(nullHandler)(nil, nil)

	assert.Equal(t, []int{1}, log)
}

func TestAddMultipleMiddleware(t *testing.T) {
	log := make([]int, 0)

	m1 := func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			log = append(log, 1)
			next(w, r)
			return nil
		}
	}
	m2 := func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			log = append(log, 2)
			next(w, r)
			return nil
		}
	}

	New(testErrorHandler(t)).Add(m1).Add(m2).ToMiddleware()(nullHandler)(nil, nil)

	assert.Equal(t, []int{1, 2}, log)
}

func TestResolve(t *testing.T) {
	log := make([]int, 0)

	m1 := func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			log = append(log, 1)
			next(w, r)
			return nil
		}
	}
	m2 := func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			log = append(log, 2)
			next(w, r)
			return nil
		}
	}
	handler := func(w http.ResponseWriter, r *http.Request) error {
		log = append(log, 3)
		return nil
	}

	New(testErrorHandler(t)).Add(m1).Add(m2).Resolve(handler)(nil, nil)

	assert.Equal(t, []int{1, 2, 3}, log)
}
