package logging

import (
	"log"

	raven "github.com/getsentry/raven-go"
)

// Logger describes the interface for the loggers in this package.
// Use Info for informational logs (including errors that are silenced).
// Use Error for errors.
type Logger interface {
	Info(string)
	Error(error)
}

// StandardLogger is a logger which proxies logs to its internal logger.
type StandardLogger struct {
	Logger *log.Logger
}

func NewStandardLogger(logger *log.Logger) StandardLogger {
	return StandardLogger{Logger: logger}
}

func (s StandardLogger) Info(msg string) {
	s.Logger.Print(msg)
}

func (s StandardLogger) Error(err error) {
	s.Logger.Printf(err.Error())
}

// SentryLogger is a logger which logs everything to its internal logger and
// additionally reports errors to Sentry.
type SentryLogger struct {
	Logger *log.Logger
	Client *raven.Client
}

func NewSentryLogger(logger *log.Logger, dsn string) (SentryLogger, error) {
	client, err := raven.New(dsn)
	if err != nil {
		return SentryLogger{}, err
	}

	return SentryLogger{
		Client: client,
		Logger: logger,
	}, nil
}

func (s SentryLogger) Info(msg string) {
	s.Logger.Print(msg)
}

func (s SentryLogger) Error(err error) {
	s.Logger.Printf(err.Error())
	s.Client.CaptureError(err, map[string]string{})
}
