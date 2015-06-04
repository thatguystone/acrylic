package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
)

type actions struct{}

func main() {
	acts := actions{}

	app := cli.NewApp()
	app.Name = "acrylic"
	app.Usage = "generate a static site"
	app.Version = "0.1"
	app.Action = acts.build

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: "_config.yml",
			Usage: "config file to use",
		},
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name:   "build",
			Usage:  "regenerate the current site",
			Action: acts.build,
		},
		cli.Command{
			Name:   "new",
			Usage:  "create a new site in the current directory",
			Action: acts.new,
		},
		cli.Command{
			Name:   "serve",
			Usage:  "serve the current site, rebuilding when changes are made",
			Action: acts.serve,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "no-watch",
					Usage: "don't monitor for changes",
				},
			},
		},
	}

	app.Run(os.Args)
}

func (actions) build(c *cli.Context) {
	err := cmdBuild(c.GlobalString("config"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (actions) new(c *cli.Context) {
	err := cmdNew(c.GlobalString("config"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (actions) serve(c *cli.Context) {

}
