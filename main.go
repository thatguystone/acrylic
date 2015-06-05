package main

import (
	"log"
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
			Name:   "clean",
			Usage:  "clean up the public dir",
			Action: acts.clean,
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
		},
	}

	// log.SetFlags(log.LstdFlags | log.Lshortfile)

	app.Run(os.Args)
}

func (actions) build(c *cli.Context) {
	err := cmdBuild(mustLoadConfig(c))
	if err != nil {
		log.Fatal(err)
	}
}

func (actions) clean(c *cli.Context) {
	cmdClean(mustLoadConfig(c))
}

func (actions) new(c *cli.Context) {
	cmdNew(c.GlobalString("config"))
}

func (actions) serve(c *cli.Context) {
	cmdServe(c.GlobalString("config"))
}
