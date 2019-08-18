package main

import (
	"time"

	"github.com/hamba/cmd"
	"github.com/hamba/pkg/log"
	"github.com/nrwiersma/proxy"
	"gopkg.in/urfave/cli.v2"
)

func runServer(c *cli.Context) error {
	ctx, err := cmd.NewContext(c)
	if err != nil {
		return err
	}

	cfg, err := newConfig(ctx)
	if err != nil {
		log.Fatal(ctx, err)
	}

	svc, err := proxy.NewServiceFromConfig(ctx, cfg)
	if err != nil {
		log.Fatal(ctx, err)
	}
	defer svc.Close()

	<-cmd.WaitForSignals()

	if err := svc.Shutdown(time.Second); err != nil {
		log.Error(ctx, "proxy: error shutting down proxy", "error", err)
	}

	return nil
}
