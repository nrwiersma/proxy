package middleware

import (
	"context"
	"strconv"

	"github.com/hamba/pkg/stats"
	"github.com/hamba/timex/mono"
	"github.com/nrwiersma/proxy/http"
)

type Stats struct {
	h     http.Handler
	stats stats.Statter
}

func NewStats(h http.Handler, s stats.Statter) *Stats {
	return &Stats{
		h:     h,
		stats: s,
	}
}

func (s *Stats) ServeHTTP(ctx context.Context, r *http.Request) *http.Response {
	start := mono.Now()
	resp := s.h.ServeHTTP(ctx, r)
	dur := mono.Since(start)

	status := strconv.Itoa(resp.StatusCode)
	tags := []string{"status", status, "status-group", string(status[0]) + "xx"}
	s.stats.Timing("request.time", dur, 1.0, tags...)
	s.stats.Inc("request.count", 1, 1.0, tags...)

	return resp
}
