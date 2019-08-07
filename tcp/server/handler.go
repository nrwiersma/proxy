package server

import (
	"context"
)

// Handler represents a handler of TCP requests.
type Handler interface {
	ServeTCP(context.Context, Conn)
}

// HandlerFunc is an adapter to allow the use of functions as TCP handlers.
type HandlerFunc func(context.Context, Conn)

// ServeTCP serves a TCP connection.
func (h HandlerFunc) ServeTCP(ctx context.Context, conn Conn) {
	h(ctx, conn)
}
