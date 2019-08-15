package router

import (
	"context"
	"net"
	"strings"
	"sync"

	"github.com/nrwiersma/proxy/http"
)

// Route is a route.
type Route struct {
	Pattern string
	Handler http.Handler
}

// Match determines if the route matches the given host and path.
func (r *Route) Match(host, path string) bool {
	s := path
	if !strings.HasPrefix(r.Pattern, "/") {
		s = host + path
	}

	return strings.HasPrefix(s, r.Pattern)
}

// Router is an HTTP request router.
//
// Routes are matched in the order they are added.
type Router struct {
	routes []*Route

	mu sync.RWMutex
}

// AddHandler adds a route handler.
//
// In order to match a path, the pattern must start with a "/" (ie /foo/bar),
// otherwise host or host and path will be matched (ie example.com or example.com/foo/bar).
func (r *Router) AddHandler(pattern string, h http.Handler) {
	r.mu.Lock()

	r.routes = append(r.routes, &Route{
		Pattern: pattern,
		Handler: h,
	})

	r.mu.Unlock()
}

// ServeHTTP serves an HTTP request.
func (r *Router) ServeHTTP(ctx context.Context, req *http.Request) *http.Response {
	r.mu.RLock()

	path := req.URL.Path
	host := r.cleanHost(req.Host)

	for _, route := range r.routes {
		if route.Match(host, path) {
			r.mu.RUnlock()
			return route.Handler.ServeHTTP(ctx, req)
		}
	}

	r.mu.RUnlock()

	return &http.Response{StatusCode: 404, StatusText: "Not Found"}
}

func (r *Router) cleanHost(host string) string {
	// Fast path
	if strings.IndexByte(host, ':') == -1 {
		return host
	}

	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}

	return host
}
