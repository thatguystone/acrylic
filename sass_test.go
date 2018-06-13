package acrylic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/cog/check"
)

func hit(h http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	return rr
}

func TestSassBasic(t *testing.T) {
	c := check.New(t)

	tmp := internal.NewTmpDir(c, map[string]string{
		"all.scss":         `@import "sub"; @import "sub2";`,
		"more/_sub.scss":   `.sub {color: #000;}`,
		"more2/_sub2.scss": `.sub2 {color: #fff;}`,
	})
	defer tmp.Remove()

	sass := NewSass(SassConfig{
		Entries: []string{
			tmp.Path("all.scss"),
		},
		IncludePaths: []string{
			tmp.Path("more"),
			tmp.Path("more2"),
		},
		Logf: c.Logf,
	})

	rr := hit(sass)
	c.Equal(rr.Code, http.StatusOK)

	body := rr.Body.String()
	c.Contains(body, `.sub {`)
	c.Contains(body, `.sub2 {`)
}

func TestSassChange(t *testing.T) {
	c := check.New(t)

	tmp := internal.NewTmpDir(c, map[string]string{
		"all.scss": `.all {color: #000;}`,
	})
	defer tmp.Remove()

	w := NewWatch(tmp.Path("."))
	defer w.Stop()

	sass := NewSass(SassConfig{
		Entries: []string{
			tmp.Path("all.scss"),
		},
		Logf: c.Logf,
	})

	w.Notify(sass)

	rr := hit(sass)
	c.Equal(rr.Code, http.StatusOK)

	tmp.WriteFile("all.scss", `.some {color: #000;}`)
	c.Until(time.Second, func() bool {
		rr := hit(sass)
		c.Equal(rr.Code, http.StatusOK)

		return strings.Contains(rr.Body.String(), ".some {")
	})
}

func TestSassErrors(t *testing.T) {
	c := check.New(t)

	tmp := internal.NewTmpDir(c, map[string]string{
		"all.scss": `@import "`,
	})
	defer tmp.Remove()

	sass := NewSass(SassConfig{
		Entries: []string{
			tmp.Path("all.scss"),
		},
		Logf: c.Logf,
	})

	rr := hit(sass)
	c.Equal(rr.Code, http.StatusInternalServerError)
}
