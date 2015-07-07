package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"

	"gopkg.in/yaml.v2"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	for run(os.Args[1:], ".", os.Stderr) {
	}
}

func run(args []string, baseDir string, logOut io.Writer) bool {
	cfg := newConfig()

	for _, arg := range args {
		b, err := ioutil.ReadFile(arg)
		if err != nil {
			panic(fmt.Errorf("failed to read config file: %v", err))
		}

		err = yaml.Unmarshal(b, cfg)
		if err != nil {
			panic(fmt.Errorf("failed to parse %s: %v", arg, err))
		}
	}

	s := site{
		args:    args,
		cfg:     cfg,
		logOut:  logOut,
		baseDir: baseDir,
	}

	if cfg.Watch {
		s.buildAndWatch()
	} else {
		ok := s.build()
		if !ok {
			os.Exit(1)
		}
	}

	return cfg.Watch
}
