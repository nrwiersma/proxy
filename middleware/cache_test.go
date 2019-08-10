package middleware_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/url"
	"testing"
	"time"

	"github.com/nrwiersma/proxy/http"
	"github.com/nrwiersma/proxy/middleware"
	"github.com/stretchr/testify/assert"
)

func TestCache_ServeHTTP(t *testing.T) {
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

	count := 0
	cache := middleware.NewCache(http.HandlerFunc(func(ctx context.Context, r *http.Request) *http.Response {
		count++

		return &http.Response{
			StatusCode: 200,
			StatusText: "OK",
			Body:       bytes.NewReader([]byte("test")),
		}
	}), middleware.CacheOpts{
		Expiry: time.Second,
		Purge:  time.Second,
	})

	// Caching run
	resp := cache.ServeHTTP(context.Background(), req)

	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, []byte("test"), body)
	resp.Body = nil
	want := &http.Response{
		StatusCode: 200,
		StatusText: "OK",
	}
	assert.Equal(t, want, resp)

	// Get cache run
	resp = cache.ServeHTTP(context.Background(), req)

	body, _ = ioutil.ReadAll(resp.Body)
	assert.Equal(t, []byte("test"), body)
	resp.Body = nil
	assert.Equal(t, want, resp)

	assert.Equal(t, 1, count)
}

func TestCache_ServeHTTPRespectsHeaders(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/test"},
		Host:   "localhost",
		Proto:  "HTTP/1.1",
		Header: http.Header{
			"Content-Type":  []string{"text/plain"},
			"Cache-Control": []string{"No-Cache"},
		},
		Body: nil,
	}

	count := 0
	cache := middleware.NewCache(http.HandlerFunc(func(ctx context.Context, r *http.Request) *http.Response {
		count++

		return &http.Response{
			StatusCode: 200,
			StatusText: "OK",
			Body:       bytes.NewReader([]byte("test")),
		}
	}), middleware.CacheOpts{
		Expiry: time.Second,
		Purge:  time.Second,
	})

	// Caching run
	_ = cache.ServeHTTP(context.Background(), req)

	// Get cache run
	_ = cache.ServeHTTP(context.Background(), req)

	assert.Equal(t, 2, count)
}

func TestCache_ServeHTTPIgnoresHeaders(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/test"},
		Host:   "localhost",
		Proto:  "HTTP/1.1",
		Header: http.Header{
			"Content-Type":  []string{"text/plain"},
			"Cache-Control": []string{"No-Cache"},
		},
		Body: nil,
	}

	count := 0
	cache := middleware.NewCache(http.HandlerFunc(func(ctx context.Context, r *http.Request) *http.Response {
		count++

		return &http.Response{
			StatusCode: 200,
			StatusText: "OK",
			Body:       bytes.NewReader([]byte("test")),
		}
	}), middleware.CacheOpts{
		Expiry:        time.Second,
		Purge:         time.Second,
		IgnoreHeaders: true,
	})

	// Caching run
	_ = cache.ServeHTTP(context.Background(), req)

	// Get cache run
	_ = cache.ServeHTTP(context.Background(), req)

	assert.Equal(t, 1, count)
}
