package acrylic

import (
	"fmt"
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
	asset string
}

func newWebpackHandler(asset string) *webpackHandler {
	return &webpackHandler{
		asset: asset,
	}
}

func (h *webpackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if webpackRevProxy != nil {
		webpackRevProxy.ServeHTTP(w, r)
		return
	}

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
			log.Printf("E: [webpack] %v", webpackErr)
		}
	})

	if webpackErr != nil {
		http.Error(w, webpackErr.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, h.asset)
}
