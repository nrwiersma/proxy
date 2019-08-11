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

	h, err = proxy.New("httpbin.org:80", proxy.Opts{Timeout: time.Second})
	//h, err = proxy.NewTLS("httpbin.org:443", "", "", proxy.Opts{Timeout: time.Second})
	if err != nil {
		return nil, err
	}

	h = middleware.NewCache(h, middleware.CacheOpts{
		Expiry:        time.Minute,
		Purge:         5 * time.Minute,
		IgnoreHeaders: true,
	})
	h = middleware.NewStats(h, ctx.Statter())
	h = middleware.NewLogger(h, ctx.Logger())

	return http.NewServer(h, http.Opts{
		IdleTimeout: time.Second,
		Log:         ctx.Logger(),
	})
}
