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
	"github.com/nrwiersma/proxy/tcp/server"
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

func newServer(ctx *cmd.Context) (*server.Server, error) {
	var h http.Handler
	var err error

	h, err = proxy.New("httpbin.org:80")
	//h, err = proxy.NewTLS("httpbin.org:443", "", "")
	if err != nil {
		return nil, err
	}
	h = middleware.NewStats(h, ctx.Statter())
	h = middleware.NewLogger(h, ctx.Logger())

	th := http.NewTransfer(h, ctx.Logger())

	return server.New(th, server.Opts{
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  time.Second,
		ErrorLog:     nil,
	})
}
