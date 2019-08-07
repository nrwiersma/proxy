package http

import (
	"context"
)

// Handler represents a handler of HTTP requests.
type Handler interface {
	ServeHTTP(context.Context, *Request) *Response
}

// HandlerFunc is an adapter to allow the use of functions as HTTP handlers.
type HandlerFunc func(context.Context, *Request) *Response

// ServeHTTP serves an HTTP connection.
func (h HandlerFunc) ServeHTTP(ctx context.Context, r *Request) *Response {
	return h(ctx, r)
}
