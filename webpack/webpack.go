// Package webpack implements a webpack-dev-server proxy
package webpack

//gocovr:skip-file

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"

	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/acrylic/proxy"
)

type webpack struct {
	bin     string
	port    uint16
	args    []string
	nodeEnv string
	proxy   *proxy.Proxy
	err     atomic.Value
}

// New creates a new webpack-dev-server runner and proxy
func New(opts ...Option) http.Handler {
	wp := &webpack{
		bin:     "./node_modules/.bin/webpack-dev-server",
		port:    9779,
		nodeEnv: "development",
	}

	for _, opt := range opts {
		opt.applyTo(wp)
	}

	proxy, err := proxy.New(fmt.Sprintf("http://localhost:%d", wp.port))
	if err != nil {
		panic(err)
	}

	wp.proxy = proxy

	go wp.run()

	return wp
}

func (wp *webpack) run() {
	args := wp.args
	args = append(args, "--port", fmt.Sprintf("%d", wp.port))

	cmd := exec.Command(wp.bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if wp.nodeEnv != "" {
		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("NODE_ENV=%s", wp.nodeEnv))
	}

	err := cmd.Run()
	wp.err.Store(fmt.Errorf("unexpected cmd exit: %v", err))
}

func (wp *webpack) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err, _ := wp.err.Load().(error)
	if err != nil {
		internal.HTTPError(w, err.Error(), http.StatusInternalServerError)
	} else {
		wp.proxy.ServeHTTP(w, r)
	}
}
