package acrylic

//gocovr:skip-file

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
)

type WebpackConfig struct {
	Bin  string   // Defaults to "./node_modules/.bin/webpack-dev-server"
	Port uint16   // Defaults to 9779
	Args []string // Extra args to pass to webpack-dev-server
}

type webpack struct {
	proxy *Proxy
	err   atomic.Value
}

func NewWebpack(cfg WebpackConfig) http.Handler {
	if cfg.Bin == "" {
		cfg.Bin = "./node_modules/.bin/webpack-dev-server"
	}

	if cfg.Port == 0 {
		cfg.Port = 9779
	}

	proxy, err := NewProxy(fmt.Sprintf("http://localhost:%d", cfg.Port))
	if err != nil {
		panic(err)
	}

	wp := webpack{
		proxy: proxy,
	}

	go wp.run(cfg)

	return &wp
}

func (wp *webpack) run(cfg WebpackConfig) {
	args := cfg.Args
	args = append(args, "--port", fmt.Sprintf("%d", cfg.Port))

	cmd := exec.Command(cfg.Bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	wp.err.Store(fmt.Errorf("unexpected cmd exit: %v", err))
}

func (wp *webpack) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err, _ := wp.err.Load().(error)
	if err != nil {
		HTTPError(w, err.Error(), http.StatusInternalServerError)
	} else {
		wp.proxy.ServeHTTP(w, r)
	}
}
