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

type crawlState struct {
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

func newState(args Args) *crawlState {
	state := &crawlState{
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
func (state *crawlState) crawl() {
	state.loadUnused()
	state.hitEntries()

	if !state.failed {
		state.deleteUnused()
	}

	if state.failed {
		panic("build failed; see previous errors")
	}
}

func (state *crawlState) loadUnused() {
	output := filepath.Clean(state.Output)

	err := filepath.Walk(output,
		func(path string, info os.FileInfo, err error) error {
			path = strings.TrimPrefix(path, output)
			if path != "" {
				state.unused[path] = info
			}

			return nil
		})
	cog.Must(err, "failed to walk existing")
}

func (state *crawlState) deleteUnused() {
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
			state.Errorf("failed to remove %s from output: %v", path, err)
		}
	}
}

func (state *crawlState) setUsed(path string) {
	for len(path) > 1 {
		delete(state.unused, path)
		path = filepath.Dir(path)
	}
}

// All crawls have to start someone. This one starts at the entry points.
func (state *crawlState) hitEntries() {
	defer state.wg.Wait()

	for _, entry := range state.EntryPoints {
		c := state.load(entry)
		if c.typ == contentExternal {
			state.Errorf("[crawl] entry point `%s` is not an internal URL",
				entry)
		}
	}
}

// Load a piece of content from the given URL. If the content is already
// loaded, it returns the existing content.
func (state *crawlState) load(url string) *content {
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
func (state *crawlState) claim(c *content, impliedPath string) bool {
	claim := func(path string) bool {
		existing, ok := state.claims[path]
		if ok {
			state.Errorf("[content] output conflict: both %s and %s use %s",
				c, existing, path)
		}

		return !ok
	}

	state.mtx.Lock()
	defer state.mtx.Unlock()

	path := filepath.Clean(c.url.Path)
	impliedPath = filepath.Join(path, impliedPath)

	if !claim(path) || !claim(impliedPath) {
		return false
	}

	state.setUsed(path)
	state.claims[path] = c

	state.setUsed(impliedPath)
	state.claims[impliedPath] = c

	return true
}

// Errorf logs an error to the user and fails the crawl
func (state *crawlState) Errorf(msg string, args ...interface{}) {
	state.failed = true
	state.Logf("E: "+msg, args...)
}

// Logf logs a message
func (state *crawlState) Logf(msg string, args ...interface{}) {
	state.Args.Logf(msg, args...)
}

// Don't need to go to the network for this. Just route directly into our
// handler.
func (state *crawlState) RoundTrip(r *http.Request) (*http.Response, error) {
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
func (state *crawlState) checkRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
