package acrylic

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/thatguystone/cog"
)

const envWebpack = "ACRYLIC_WEBPACK"

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
	asset    string
	revProxy *httputil.ReverseProxy
}

func newWebpackHandler(asset string) (h webpackHandler) {
	h.asset = asset

	remote := os.Getenv(envWebpack)
	if remote != "" {
		h.revProxy = httputil.NewSingleHostReverseProxy(&url.URL{
			Scheme: "http",
			Host:   remote,
		})
	}

	return
}

func (h webpackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.revProxy != nil {
		h.revProxy.ServeHTTP(w, r)
	} else {
		http.ServeFile(w, r, h.asset)
	}
}
