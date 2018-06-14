package sass

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thatguystone/acrylic/internal/testutil"
	"github.com/thatguystone/acrylic/watch"
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

	tmp := testutil.NewTmpDir(c, map[string]string{
		"all.scss":         `@import "sub"; @import "sub2";`,
		"more/_sub.scss":   `.sub {color: #000;}`,
		"more2/_sub2.scss": `.sub2 {color: #fff;}`,
	})
	defer tmp.Remove()

	sass := New(
		tmp.Path("all.scss"),
		IncludePaths(
			tmp.Path("more"),
			tmp.Path("more2")),
		LogTo(c.Logf))

	rr := hit(sass)
	c.Equal(rr.Code, http.StatusOK)

	body := rr.Body.String()
	c.Contains(body, `.sub {`)
	c.Contains(body, `.sub2 {`)
}

func TestSassChange(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, map[string]string{
		"all.scss": `.all {color: #000;}`,
	})
	defer tmp.Remove()

	w := watch.New(tmp.Path("."))
	defer w.Stop()

	sass := New(
		tmp.Path("all.scss"),
		LogTo(c.Logf),
		Watch(w))

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

	tmp := testutil.NewTmpDir(c, map[string]string{
		"all.scss": `@import "`,
	})
	defer tmp.Remove()

	sass := New(
		tmp.Path("all.scss"),
		LogTo(c.Logf))

	rr := hit(sass)
	c.Equal(rr.Code, http.StatusInternalServerError)
}
