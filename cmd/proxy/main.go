package main

import (
	"log"
	"os"

	"github.com/hamba/cmd"
	"gopkg.in/urfave/cli.v2"
)

import _ "github.com/joho/godotenv/autoload"

const (
	flagTemplates = "templates"
)

var version = "¯\\_(ツ)_/¯"

var commands = []*cli.Command{
	{
		Name:  "server",
		Usage: "Run the reverse proxy",
		Flags: cmd.Flags{
			//&cli.StringFlag{
			//	Name:    flagTemplates,
			//	Value:   "file:///./templates",
			//	Usage:   "The URI to the templates. Supported schemes: 'file', 'http', 'https'.",
			//	EnvVars: []string{"TEMPLATES"},
			//},
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
