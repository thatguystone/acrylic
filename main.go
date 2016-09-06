package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	for run(os.Args[1:], ".", false) {
	}
}

func run(args []string, baseDir string, testing bool) bool {
	cfg := config.NewC()
	err := cfg.Load(args...)
	err.Must(err)

	build := func() bool {
		s := site{
			args:    args,
			cfg:     cfg,
			log:     log,
			baseDir: baseDir,
		}

		return s.build()
	}

	if cfg.Debug && !testing {
		buildWatchAndServe(build, cfg, log, args, baseDir)
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

	log.Printlin("Serving on %s ...\n", cfg.DebugAddr)

	fnot := newFnotify()
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
			log.Println("Change detected, rebuilding...")
			build()
		}
	}
}
