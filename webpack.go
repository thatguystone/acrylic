package acrylic

//gocovr:skip-file

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Webpack runs webpack-dev-server and proxies requests to it
type Webpack struct {
	Bin   string   // Defaults to "./node_modules/.bin/webpack-dev-server"
	Port  uint16   // Defaults to 8779
	Args  []string // Extra args to pass to webpack-dev-server
	once  sync.Once
	proxy Proxy
	err   atomic.Value
}

// Webpack is started on demand; this allows you to start it early to kick off
// builds as soon as possible.
func (wp *Webpack) Start() {
	wp.once.Do(func() {
		wp.init()
		go wp.run()
	})
}

func (wp *Webpack) init() {
	if wp.Bin == "" {
		wp.Bin = "./node_modules/.bin/webpack-dev-server"
	}

	if wp.Port == 0 {
		wp.Port = 8779
	}

	wp.proxy.To = fmt.Sprintf("http://localhost:%d", wp.Port)
}

func (wp *Webpack) run() {
	args := wp.Args
	args = append(args, "--port", fmt.Sprintf("%d", wp.Port))

	cmd := exec.Command(wp.Bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	wp.err.Store(fmt.Errorf("unexpected cmd exit: %v", err))
}

func (wp *Webpack) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wp.Start()

	err, _ := wp.err.Load().(error)
	wp.proxy.Serve(err, w, r)
}
