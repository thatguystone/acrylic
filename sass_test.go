package acrylic

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rjeczalik/notify"
	"github.com/thatguystone/cog/check"
)

func (sass *Sass) hit() *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	sass.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	return rr
}

type eventInfo struct {
	path string
}

func (ev eventInfo) Event() notify.Event { return 0 }
func (ev eventInfo) Path() string        { return ev.path }
func (ev eventInfo) Sys() interface{}    { return nil }

func TestSassBasic(t *testing.T) {
	c := check.New(t)

	fs, cleanup := c.FS()
	defer cleanup()

	fs.SWriteFile("all.scss", `@import "sub"; @import "sub2";`)
	fs.SWriteFile("recurse/all.scss", `.recurse {color:#0f0;}`)
	fs.SWriteFile("more/_sub.scss", `.sub {color: #000;}`)
	fs.SWriteFile("more2/_sub2.scss", `.sub2 {color: #fff;}`)

	sass := Sass{
		Entries: []string{
			fs.Path("all.scss"),
		},
		IncludePaths: []string{
			fs.Path("more"),
			fs.Path("more2"),
		},
		Recurse: []string{
			fs.Path("recurse"),
		},
	}

	c.Nil(sass.rebuild())
	c.Contains(sass.compiled.String(), ".sub {")
	c.Contains(sass.compiled.String(), ".recurse {")
}

func TestSassServeAndChange(t *testing.T) {
	c := check.New(t)

	fs, cleanup := c.FS()
	defer cleanup()

	fs.SWriteFile("all.scss", `.all {color: #000;}`)

	sass := Sass{
		Entries: []string{
			fs.Path("all.scss"),
		},
	}

	rr := sass.hit()
	c.Equal(rr.Code, http.StatusOK)
	c.Contains(rr.Body.String(), ".all {")

	fs.SWriteFile("all.scss", `.some {color: #000;}`)
	sass.Changed(WatchEvents{
		eventInfo{path: fs.Path("all.scss")},
	})

	c.Until(time.Second, func() bool {
		rr = sass.hit()
		return strings.Contains(rr.Body.String(), ".some {")
	})
}

func TestSassErrors(t *testing.T) {
	c := check.New(t)

	fs, cleanup := c.FS()
	defer cleanup()

	fs.SWriteFile("all.scss", `@import "`)

	sass := Sass{
		Entries: []string{
			fs.Path("all.scss"),
		},
		Recurse: []string{
			fs.Path("doesnt exist"),
		},
	}
	c.Equal(sass.hit().Code, http.StatusInternalServerError)

	sass = Sass{
		Entries: []string{
			fs.Path("all.scss"),
		},
	}
	c.Equal(sass.hit().Code, http.StatusInternalServerError)

	err := sass.updateLastMod([]string{"/does/not/exist"})
	c.True(os.IsNotExist(err))
}
