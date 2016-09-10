package crawl

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/thatguystone/cog"
	"github.com/thatguystone/cog/cync"
)

type state struct {
	Args

	failed bool

	mtx    sync.Mutex
	unused map[string]os.FileInfo
	loaded map[string]*content
	claims map[string]*content

	wg         sync.WaitGroup
	httpClient http.Client
	sema       *cync.Semaphore
}

func newState(args Args) *state {
	state := &state{
		Args: args,

		unused: map[string]os.FileInfo{},
		loaded: map[string]*content{},
		claims: map[string]*content{},
		sema:   cync.NewSemaphore(runtime.NumCPU() + 1),
	}

	state.httpClient.Transport = state
	state.httpClient.CheckRedirect = state.checkRedirect

	return state
}

// Run the full crawl
func (state *state) crawl() {
	state.loadUnused()
	state.hitEntries()

	if !state.failed {
		state.deleteUnused()
	}

	if state.failed {
		panic("[crawl] build failed; see previous errors")
	}
}

func (state *state) loadUnused() {
	output := filepath.Clean(state.Output)

	err := filepath.Walk(output,
		func(path string, info os.FileInfo, err error) error {
			path = strings.TrimPrefix(path, output)
			if path != "" {
				state.unused[path] = info
			}

			return nil
		})
	cog.Must(err, "[crawl] failed to walk existing")
}

func (state *state) deleteUnused() {
	paths := make([]string, 0, len(state.unused))
	for p := range state.unused {
		paths = append(paths, p)
	}

	// Sort in reverse so that directories come after files
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))

	for _, path := range paths {
		outPath := filepath.Join(state.Output, path)
		err := os.Remove(outPath)
		if err != nil {
			state.Errorf("[crawl] "+
				"failed to remove %s from output: %v",
				path, err)
		}
	}
}

func (state *state) setUsed(path string) {
	for len(path) > 1 {
		delete(state.unused, path)
		path = filepath.Dir(path)
	}
}

// All crawls have to start someone. This one starts at the entry points.
func (state *state) hitEntries() {
	defer state.wg.Wait()

	for _, entry := range state.EntryPoints {
		c := state.load(entry)
		if c.typ == contentExternal {
			state.Errorf("[crawl] "+
				"entry point `%s` is not an internal URL",
				entry)
		}
	}
}

// Load a piece of content from the given URL. If the content is already
// loaded, it returns the existing content.
func (state *state) load(url string) *content {
	state.mtx.Lock()
	defer state.mtx.Unlock()

	c, ok := state.loaded[url]
	if !ok {
		c = newContent(state, url)
		state.loaded[url] = c
	}

	return c
}

// Claim the output path for the given content
func (state *state) claim(
	c *content,
	paths []string) (oc *content, oPath string, ok bool) {

	claim := func(path string) bool {
		oc = state.claims[path]
		oPath = path
		return oc == nil || oc == c
	}

	state.mtx.Lock()
	defer state.mtx.Unlock()

	for i, path := range paths {
		path = filepath.Clean(path)
		paths[i] = path

		if !claim(path) {
			return
		}
	}

	for _, path := range paths {
		state.setUsed(path)
		state.claims[path] = c
	}

	ok = true
	return
}

// Errorf logs an error to the user and fails the crawl
func (state *state) Errorf(msg string, args ...interface{}) {
	state.failed = true
	state.Logf("E: "+msg, args...)
}

// Logf logs a message
func (state *state) Logf(msg string, args ...interface{}) {
	state.Args.Logf(msg, args...)
}

// Don't need to go to the network for this. Just route directly into our
// handler.
func (state *state) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()

	// Don't pound the handler: it might be doing image scaling, video
	// scaling, etc, out of process, so give it a break.
	state.sema.Lock()
	defer state.sema.Unlock()

	state.Handler.ServeHTTP(w, r)

	resp := w.Result()
	resp.Request = r

	return resp, nil
}

// Redirects are disabled. They're handled explicitly by content.
func (state *state) checkRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
