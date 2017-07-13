package client

import (
	"bytes"
	"errors"
	"io"
)

// ErrorFromReader takes any Reader and returns an error
// with the contents of the Reader.
// A quick and dirty way to turn API responses into errors.
func ErrorFromReader(r io.Reader) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	return errors.New(buf.String())
}
