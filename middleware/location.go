package middleware

import (
	"context"

	"github.com/nrwiersma/proxy/http"
)

// Location sets the path of a request.
type Location struct {
	h    http.Handler
	path string
}

// NewLocation returns a location setting middleware.
func NewLocation(h http.Handler, path string) *Location {
	return &Location{
		h:    h,
		path: path,
	}
}

// ServeHTTP serves an HTTP request.
func (l *Location) ServeHTTP(ctx context.Context, r *http.Request) *http.Response {
	r.URL.Path = l.path

	return l.h.ServeHTTP(ctx, r)
}
