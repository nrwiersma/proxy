package middleware

import (
	"context"

	"github.com/hamba/pkg/log"
	"github.com/nrwiersma/proxy/http"
)

// Logger logs the request.
type Logger struct {
	h   http.Handler
	log log.Logger
}

// NewLogger returns a logger middlware.
func NewLogger(h http.Handler, l log.Logger) *Logger {
	return &Logger{
		h:   h,
		log: l,
	}
}

// ServeHTTP serves an HTTP request.
func (l *Logger) ServeHTTP(ctx context.Context, r *http.Request) *http.Response {
	l.log.Info("", "request", r)

	resp := l.h.ServeHTTP(ctx, r)

	l.log.Info("", "response", resp)

	return resp
}
