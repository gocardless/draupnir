package chain

import (
	"errors"
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

	New(testErrorHandler(t)).Add(middleware).Resolve(nullHandler)(nil, nil)

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

	New(testErrorHandler(t)).Add(m1).Add(m2).Resolve(nullHandler)(nil, nil)

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

func TestErrorHandler(t *testing.T) {
	log := make([]string, 0)

	errorHandler := func(next Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := next(w, r)
			log = append(log, err.Error())
		}
	}

	m := func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			log = append(log, "middleware before")
			next(w, r)
			log = append(log, "middleware after")
			return errors.New("some error")
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) error {
		log = append(log, "handler")
		return nil
	}

	New(errorHandler).Add(m).Resolve(handler)(nil, nil)

	assert.Equal(
		t,
		[]string{"middleware before", "handler", "middleware after", "some error"},
		log,
	)
}
