package acrylic

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/thatguystone/cog/check"
)

func TestCrawlNeedsFingerprint(t *testing.T) {
	c := check.New(t)

	cr := Crawl{
		Fingerprint: []string{
			"application/json",
			".js",
			".test",
		},
	}

	c.True(cr.needsFingerprint("application/json"))
	c.True(cr.needsFingerprint(".js"))
	c.True(cr.needsFingerprint(".test"))

	c.False(cr.needsFingerprint("application/js"))
	c.False(cr.needsFingerprint(".test2"))
}

func TestCrawlSaveNoChange(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		io.WriteString(w, `index`)
	})

	cr := Crawl{
		Handler: r,
		Output:  fs.Path("/dst"),
		Logf:    c.Logf,
	}
	c.Nil(cr.Do())

	first, err := os.Stat(fs.Path("/dst/index.html"))
	c.Must.Nil(err)

	cr = Crawl{
		Handler: r,
		Output:  fs.Path("/dst"),
		Logf:    c.Logf,
	}
	c.Nil(cr.Do())

	second, err := os.Stat(fs.Path("/dst/index.html"))
	c.Must.Nil(err)

	c.Equal(first.ModTime(), second.ModTime())
}

type readSeeker struct {
	er *check.Errorer
}

func (r readSeeker) Read(b []byte) (n int, err error) {
	err = r.er.Err()
	if err == nil {
		err = io.EOF
	}

	return
}

func (r readSeeker) Seek(offset int64, whence int) (n int64, err error) {
	err = r.er.Err()
	return
}

func TestCrawlSaveErrors(t *testing.T) {
	c := check.New(t)
	cr := Crawl{}
	rs := readSeeker{er: new(check.Errorer)}

	fs, clean := c.FS()
	defer clean()

	fs.SWriteFile("/test/file", "file")

	err := cr.save(fs.Path("/test"), strings.NewReader(""))
	c.NotNil(err)

	err = cr.save("/totally/not/allowed", strings.NewReader(""))
	c.NotNil(err)

	check.UntilNil(10, func() error {
		return cr.save(fs.Path("/test/file"), rs)
	})
}

func TestCrawlCoverage(t *testing.T) {
	c := check.New(t)

	cr := Crawl{}
	cr.init()

	c.NotNil(cr.Logf)
}
