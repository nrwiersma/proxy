package router_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/nrwiersma/proxy/http"
	"github.com/nrwiersma/proxy/http/router"
	"github.com/stretchr/testify/assert"
)

func TestRoute_Match(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		host    string
		path    string
		want    bool
	}{
		{
			name:    "Match Path",
			pattern: "/foo/bar",
			host:    "example.com",
			path:    "/foo/bar/baz/bat",
			want:    true,
		},
		{
			name:    "Match Host And Path",
			pattern: "example.com/foo/bar",
			host:    "example.com",
			path:    "/foo/bar/baz/bat",
			want:    true,
		},
		{
			name:    "No Match Path",
			pattern: "/foo/bar",
			host:    "example.com",
			path:    "/something/bar/baz/bat",
			want:    false,
		},
		{
			name:    "No Match Host And Path",
			pattern: "example.com/foo/bar",
			host:    "example.com",
			path:    "/something/bar/baz/bat",
			want:    false,
		},

		{
			name:    "No Match Host",
			pattern: "something.com/foo/bar",
			host:    "example.com",
			path:    "/foo/bar/baz/bat",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &router.Route{Pattern: tt.pattern}

			got := r.Match(tt.host, tt.path)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRouter_ServeHTTP(t *testing.T) {
	h1 := http.HandlerFunc(func(_ context.Context, r *http.Request) *http.Response {
		return &http.Response{StatusCode: 200}
	})
	h2 := http.HandlerFunc(func(_ context.Context, r *http.Request) *http.Response {
		return &http.Response{StatusCode: 204}
	})

	r := &router.Router{}
	r.AddHandler("example.com/foo/bar", h1)
	r.AddHandler("example.com/foo/bar/bat", h2)

	req := &http.Request{
		URL:  &url.URL{Path: "/foo/bar/bat/baz"},
		Host: "example.com:8080",
	}

	got := r.ServeHTTP(context.Background(), req)

	assert.Equal(t, 200, got.StatusCode)
}

func TestRouter_ServeHTTPNotFoundReturns404(t *testing.T) {
	h1 := http.HandlerFunc(func(_ context.Context, r *http.Request) *http.Response {
		return &http.Response{StatusCode: 200}
	})
	h2 := http.HandlerFunc(func(_ context.Context, r *http.Request) *http.Response {
		return &http.Response{StatusCode: 204}
	})

	r := &router.Router{}
	r.AddHandler("example.com/foo/bar", h1)
	r.AddHandler("example.com/foo/bar/bat", h2)

	req := &http.Request{
		URL:  &url.URL{Path: "/foo/bar/bat/baz"},
		Host: "something.com",
	}

	got := r.ServeHTTP(context.Background(), req)

	assert.Equal(t, 404, got.StatusCode)
}
