package main

import (
	"log"
	"os"
	"runtime/pprof"

	"github.com/codegangsta/cli"
)

type actions struct{}

func main() {
	if runtime.GOMAXPROCS(-1) == 1 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

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
			Name:   "benchmark",
			Usage:  "benchmark by building the docs repeatedly",
			Action: acts.benchmark,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "times",
					Value: 1000,
					Usage: "how many times to build the docs",
				},
			},
		},
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

func (actions) benchmark(c *cli.Context) {
	cfg, err := loadConfig("docs/_config.yml")
	if err != nil {
		log.Fatal(err)
	}

	cpuProfile := "cpu.out"
	f, err := os.Create(cpuProfile)
	if err != nil {
		log.Fatal(err)
	}

	pprof.StartCPUProfile(f)
	defer f.Close()
	defer pprof.StopCPUProfile()

	cfg.hideStats = true
	times := c.Int("times")

	for i := 0; i < times; i++ {
		err := cmdBuild(cfg)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("CPU profile written to", cpuProfile)
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
