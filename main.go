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

	for i := len(args) - 1; i >= 0; i-- {
		b, err := ioutil.ReadFile(args[i])
		if err != nil {
			panic(fmt.Errorf("failed to read config file: %v", err))
		}

		err = yaml.Unmarshal(b, cfg)
		if err != nil {
			panic(fmt.Errorf("failed to parse %s: %v", args[i], err))
		}
	}

	s := site{
		args:    args,
		cfg:     cfg,
		logOut:  logOut,
		baseDir: baseDir,
	}

	if cfg.Debug {
		s.buildAndWatch()
	} else {
		s.build()
	}

	return cfg.Debug
}
