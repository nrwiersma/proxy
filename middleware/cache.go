package middleware

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/nrwiersma/proxy/http"
	"github.com/nrwiersma/proxy/internal/slices"
	"github.com/patrickmn/go-cache"
)

var (
	cacheControl        = "Cache-Control"
	cacheControlNoCache = []string{
		"no-cache",
		"no-store",
		"private",
	}
)

// Cache caches http responses.
type Cache struct {
	h     http.Handler
	cache *cache.Cache

	ignoreHeaders bool
}

type CacheOpts struct {
	Expiry time.Duration

	Purge time.Duration

	IgnoreHeaders bool
}

// NewCache returns a cache middleware.
func NewCache(h http.Handler, opts CacheOpts) *Cache {
	c := cache.New(opts.Expiry, opts.Purge)

	return &Cache{
		h:             h,
		cache:         c,
		ignoreHeaders: opts.IgnoreHeaders,
	}
}

type cacheItem struct {
	Response http.Response
	Body     []byte
}

// ServeHTTP serves an HTTP request.
func (c *Cache) ServeHTTP(ctx context.Context, r *http.Request) *http.Response {
	key := c.cacheKey(r)
	if v, ok := c.cache.Get(key); ok {
		item := v.(cacheItem)

		newResp := &http.Response{}
		*newResp = item.Response
		newResp.Body = bytes.NewReader(item.Body)
		return newResp
	}

	resp := c.h.ServeHTTP(ctx, r)

	if !c.shouldCache(r, resp) {
		return resp
	}

	body, err := c.readBody(resp)
	if err != nil {
		return resp
	}

	newResp := *resp
	newResp.Body = nil

	c.cache.Set(key, cacheItem{
		Response: newResp,
		Body:     body,
	}, cache.DefaultExpiration)

	return resp
}

func (c *Cache) cacheKey(req *http.Request) string {
	return req.Host + req.URL.String()
}

func (c *Cache) shouldCache(req *http.Request, resp *http.Response) bool {
	if c.ignoreHeaders {
		return true
	}

	cc := strings.ToLower(req.Header.Get(cacheControl))
	if slices.StringContains(cc, cacheControlNoCache) {
		return false
	}

	cc = strings.ToLower(resp.Header.Get(cacheControl))
	if slices.StringContains(cc, cacheControlNoCache) {
		return false
	}

	return resp.Header.Get("Set-Cookie") == ""
}

func (c *Cache) readBody(resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return nil, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if seeker, ok := resp.Body.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err == nil {
			return body, nil
		}
	}

	resp.Body = bytes.NewReader(body)
	return body, nil
}
