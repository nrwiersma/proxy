package main

import (
	"log"
	"os"

	"github.com/hamba/cmd"
	"gopkg.in/urfave/cli.v2"
)

import _ "github.com/joho/godotenv/autoload"

const (
	flagConfig = "config"
)

var version = "¯\\_(ツ)_/¯"

var commands = []*cli.Command{
	{
		Name:  "server",
		Usage: "Run the reverse proxy",
		Flags: cmd.Flags{
			&cli.StringFlag{
				Name:    flagConfig + ",c",
				Value:   "./config.yml",
				Usage:   "The proxy configuration file.",
				EnvVars: []string{"CONFIG"},
			},
		}.Merge(cmd.CommonFlags, cmd.ServerFlags),
		Action: runServer,
	},
}

func main() {
	app := newApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newApp() *cli.App {
	return &cli.App{
		Name:     "ren",
		Version:  version,
		Commands: commands,
	}
}
