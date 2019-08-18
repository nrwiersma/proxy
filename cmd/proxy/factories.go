package main

import (
	"os"

	"github.com/hamba/cmd"
	"github.com/nrwiersma/proxy"
)

func newConfig(c *cmd.Context) (*proxy.Config, error) {
	f, err := os.Open(c.String(flagConfig))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return proxy.ParseConfig(f)
}
