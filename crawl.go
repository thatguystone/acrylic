package acrylic

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thatguystone/cog/cfs"
)

// Crawl is used to a crawl a handler and generate a static site
type Crawl struct {
	Handler     http.Handler // Handler to crawl
	EntryPoints []string     // Entry points to crawl
	Fingerprint []string     // Content-Types and extensions to fingerprint
	Output      string       // Build directory
	Logf        func(string, ...interface{})
	wg          sync.WaitGroup
	failed      bool // If there were errors while crawling
	mtx         sync.Mutex
	content     map[string]*Content
}

const indent = "    "

var errCrawl = errors.New("crawl failed; check log for details")

// Do performs the actual crawl
func (cr *Crawl) Do() (err error) {
	cr.init()

	for _, entry := range cr.EntryPoints {
		u, err := url.Parse(path.Clean(entry))
		if err != nil {
			cr.Errorf("[crawl] invalid entry point: %s: %v", entry, err)
		} else {
			cr.getContent(u)
		}
	}

	cr.wg.Wait()

	if cr.failed {
		err = errCrawl
	}

	return
}

func (cr *Crawl) init() {
	if len(cr.EntryPoints) == 0 {
		cr.EntryPoints = []string{"/"}
	}

	if cr.Logf == nil {
		cr.Logf = log.Printf
	}
}

func (cr *Crawl) getContent(u *url.URL) *Content {
	c, existed := cr.newContent(u)
	if !existed {
		cr.wg.Add(1)
		go c.load()
	}

	return c
}

func (cr *Crawl) newContent(u *url.URL) (c *Content, alreadyExists bool) {
	uu := *u
	uu.RawQuery = ""
	uu.ForceQuery = false
	uu.Fragment = ""
	normu := uu.String()

	cr.mtx.Lock()

	if cr.content == nil {
		cr.content = map[string]*Content{}
	}

	c, alreadyExists = cr.content[normu]
	if !alreadyExists {
		c = newContent(cr, *u)
		cr.content[normu] = c
	}

	cr.mtx.Unlock()

	return
}

func (cr *Crawl) needsFingerprint(idents ...string) bool {
	for _, s := range cr.Fingerprint {
		for _, ident := range idents {
			if s == ident {
				return true
			}
		}
	}

	return false
}

func (cr *Crawl) outputPath(path string) string {
	path = addIndex(path)
	return filepath.Join(cr.Output, path)
}

func (cr *Crawl) setUsed(dst string) {

}

func (cr *Crawl) save(dst string, rs io.ReadSeeker) error {
	dstHash, err := hashFile(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	rsHash, err := hashReader(rs)
	if err != nil {
		return err
	}

	// Help out rsync: don't touch file if contents are the same
	if bytes.Equal(dstHash, rsHash) {
		return nil
	}

	_, err = rs.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	f, err := cfs.Create(dst)
	if err != nil {
		return err
	}

	defer f.Close()

	cr.setUsed(dst)
	_, err = io.Copy(f, rs)
	if err == nil {
		err = f.Close()
	}

	return err
}

// Errorf reports that an error occurred. This is exported so that `go vet` will
// check it.
func (cr *Crawl) Errorf(format string, args ...interface{}) {
	cr.failed = true
	cr.Logf(format, args...)
}

// Get gets the content at the given URL
func (cr *Crawl) Get(url string) (c *Content) {
	cr.mtx.Lock()
	c = cr.content[url]
	cr.mtx.Unlock()

	return
}

func addIndex(path string) string {
	if strings.HasSuffix(path, "/") {
		path += "index.html"
	}

	return path
}
