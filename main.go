package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v2"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	for run(os.Args[1:], ".", os.Stderr, false) {
	}
}

func run(args []string, baseDir string, logOut io.Writer, testing bool) bool {
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

	build := func() bool {
		s := site{
			args:    args,
			cfg:     cfg,
			logOut:  logOut,
			baseDir: baseDir,
		}

		return s.build()
	}

	if cfg.Debug && !testing {
		buildWatchAndServe(build, cfg, args, baseDir)
		return true
	}

	ok := build()
	if !ok && !testing {
		os.Exit(1)
	}

	return false
}

func buildWatchAndServe(
	build func() bool,
	cfg *config,
	args []string,
	baseDir string) {

	build()

	l, err := net.Listen("tcp", cfg.DebugAddr)
	if err != nil {
		panic(err)
	}

	closed := false
	defer func() {
		closed = true
		l.Close()
	}()

	server := http.Server{
		Handler: http.FileServer(http.Dir(cfg.PublicDir)),
	}

	go func() {
		err = server.Serve(l)
		if !closed && err != nil {
			panic(err)
		}
	}()

	fmt.Printf("Serving on %s ...\n", cfg.DebugAddr)

	fnot := fwatch()
	defer fnot.close()

	fnot.addDir(filepath.Join(baseDir, cfg.AssetsDir))
	fnot.addDir(filepath.Join(baseDir, cfg.ContentDir))
	fnot.addDir(filepath.Join(baseDir, cfg.DataDir))
	fnot.addDir(filepath.Join(baseDir, cfg.TemplatesDir))

	for _, arg := range args {
		fnot.addFile(arg)
	}

	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()

	for {
		select {
		case path := <-fnot.changed:
			for _, arg := range args {
				if arg == path {
					time.Sleep(10 * time.Millisecond)
					return
				}
			}

			timer.Reset(50 * time.Millisecond)

		case <-timer.C:
			fmt.Println("Change detected, rebuilding...")
			build()
		}
	}
}
