package proxy

import (
	"context"
	"sync"

	"github.com/nrwiersma/proxy/http"
)

// RRLoadBalancer is a round robin load balancer.
type RRLoadBalancer struct {
	srvs []http.Handler
	mu   sync.Mutex
	pos  int
}

// NewRRLoadBalancer returns a round robin load balancer.
func NewRRLoadBalancer(server http.Handler, servers ...http.Handler) *RRLoadBalancer {
	srvs := append([]http.Handler{server}, servers...)

	return &RRLoadBalancer{
		srvs: srvs,
		pos:  0,
	}
}

// ServeHTTP serves an HTTP request.
func (b *RRLoadBalancer) ServeHTTP(ctx context.Context, r *http.Request) *http.Response {
	b.mu.Lock()

	h := b.srvs[b.pos]
	b.pos++
	if b.pos >= len(b.srvs) {
		b.pos = 0
	}

	b.mu.Unlock()

	return h.ServeHTTP(ctx, r)
}
