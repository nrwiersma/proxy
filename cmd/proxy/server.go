package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hamba/cmd"
	"github.com/hamba/pkg/log"
	"github.com/nrwiersma/proxy/http"
	"github.com/nrwiersma/proxy/http/proxy"
	"github.com/nrwiersma/proxy/middleware"
	"gopkg.in/urfave/cli.v2"
)

func runServer(c *cli.Context) error {
	ctx, err := cmd.NewContext(c)
	if err != nil {
		return err
	}

	srv, err := newServer(ctx)
	if err != nil {
		log.Fatal(ctx, err)
	}
	defer srv.Close()

	go func() {
		port := c.String(cmd.FlagPort)
		log.Info(ctx, fmt.Sprintf("Starting server on port %s", port))
		if err := srv.ListenAndServe(":" + port); err != nil {
			log.Fatal(ctx, "proxy: server error", "error", err)
		}
	}()

	<-cmd.WaitForSignals()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error(ctx, "proxy: error shutting down", "error", err)
	}
	cancel()

	return nil
}

func newServer(ctx *cmd.Context) (*http.Server, error) {
	var h http.Handler
	var err error

	srv1, err := proxy.New("127.0.0.1:9080", proxy.Opts{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	srv2, err := proxy.New("127.0.0.1:9081", proxy.Opts{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	h = proxy.NewRRLoadBalancer(srv1, srv2)

	h = middleware.NewCache(h, middleware.CacheOpts{
		Expiry:        10*time.Second,
		Purge:         time.Minute,
		IgnoreHeaders: true,
	})
	h = middleware.NewStats(h, ctx.Statter())
	h = middleware.NewLogger(h, ctx.Logger())

	return http.NewServer(h, http.Opts{
		IdleTimeout: time.Second,
		Log:         ctx.Logger(),
	})
}
