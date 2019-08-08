package middleware

import (
	"context"

	"github.com/hamba/pkg/log"
	"github.com/nrwiersma/proxy/http"
)

type Logger struct {
	h   http.Handler
	log log.Logger
}

func NewLogger(h http.Handler, lgbl log.Loggable) *Logger {
	return &Logger{
		h:   h,
		log: lgbl.Logger(),
	}
}

func (l *Logger) ServeHTTP(ctx context.Context, r *http.Request) *http.Response {
	l.log.Info("", "request", r)

	resp := l.h.ServeHTTP(ctx, r)

	l.log.Info("", "response", resp)

	return resp
}