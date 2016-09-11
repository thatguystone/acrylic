package acrylic

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/thatguystone/cog"
	"github.com/thatguystone/cog/stringc"
)

const (
	envWebpack = "ACRYLIC_WEBPACK"
	nodeBin    = "./node_modules/.bin"
)

// Only run webpack once per build
var (
	webpackRevProxy *httputil.ReverseProxy
	webpackOnce     sync.Once
	webpackErr      error
)

func init() {
	remote := os.Getenv(envWebpack)
	if remote != "" {
		webpackRevProxy = httputil.NewSingleHostReverseProxy(&url.URL{
			Scheme: "http",
			Host:   remote,
		})
	}
}

// When running webpack-dev-server, the child process needs to know about it
// so that it serves from it instead of from the FS.
func setWebpackEnv(host, port string) {
	if host == "" {
		host = "localhost"
	}

	err := os.Setenv(envWebpack, fmt.Sprintf("%s:%s", host, port))
	cog.Must(err, "failed to set "+envWebpack)
}

type webpackHandler struct {
	handler
	asset string
}

func newWebpackHandler(asset string) *webpackHandler {
	c := &webpackHandler{
		asset: asset,
	}

	// I mean, why not start as early as possible?
	if webpackRevProxy != nil {
		go c.compile()
	}

	return c
}

func (h *webpackHandler) compile() {
	webpackOnce.Do(func() {
		start := time.Now()
		defer func() {
			log.Printf("I: [webpack] build took %s", time.Now().Sub(start))
		}()

		cmd := exec.Command(nodeBin + "/webpack")
		out, err := cmd.CombinedOutput()
		if err != nil {
			webpackErr = fmt.Errorf("%v:\n%s",
				err,
				stringc.Indent(string(out), "    "))
		}
	})
}

func (h *webpackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if webpackRevProxy != nil {
		webpackRevProxy.ServeHTTP(w, r)
		return
	}

	h.compile()

	switch {
	case webpackErr != nil:
		h.errorf(w, webpackErr, "[webpack] compile failed")

	case h.needsBusted(r):
		body, err := ioutil.ReadFile(h.asset)
		if err != nil {
			h.errorf(w, err, "[webpack] failed to read file")
		} else {
			h.handler.redirectBusted(
				w, r,
				*r.URL, h.hashBuster(body))
		}

	default:
		http.ServeFile(w, r, h.asset)
	}
}
