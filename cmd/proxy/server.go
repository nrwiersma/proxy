package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hamba/cmd"
	"github.com/hamba/pkg/log"
	"github.com/nrwiersma/proxy/http"
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
	h := http.HandlerFunc(func(ctx context.Context, w io.WriteCloser, req *http.Request) {
		fmt.Printf("%#v\n", req)
		fmt.Printf("%#v\n", req.URL)
		w.Write([]byte("HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n400 Bad Request"))
		w.Close()
	})

	th := http.NewTransfer(h)

	return server.New(th, server.Opts{
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  0,
		ErrorLog:     nil,
	})
}
