package middleware_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/nrwiersma/proxy/http"
	"github.com/nrwiersma/proxy/middleware"
	"github.com/stretchr/testify/assert"
)

func TestLocation_ServeHTTP(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/test"},
		Host:   "localhost",
		Proto:  "HTTP/1.1",
		Header: http.Header{
			"Content-Type": []string{"text/plain"},
		},
		Body: nil,
	}

	loc := middleware.NewLocation(http.HandlerFunc(func(ctx context.Context, r *http.Request) *http.Response {
		assert.Equal(t, "/foobar", r.URL.Path)

		return &http.Response{StatusCode: 200, StatusText: "OK"}
	}), "/foobar")

	_ = loc.ServeHTTP(context.Background(), req)
}
