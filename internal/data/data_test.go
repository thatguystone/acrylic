package data

import (
	"os"
	"testing"
	"time"

	"github.com/thatguystone/acrylic/internal/config"
	"github.com/thatguystone/cog/check"
)

func TestBasic(t *testing.T) {
	c := check.New(t)
	d := New(config.New())

	err := d.LoadData(check.Fixture("json.json"))
	c.MustNotError(err)

	err = d.LoadData(check.Fixture("python.py"))
	c.MustNotError(err)

	c.NotEqual(d.Get("json"), nil)
	m := d.Get("json").(map[string]interface{})
	c.Equal(m["key"].(float64), 1234.0)

	c.NotEqual(d.Get("python"), nil)
	m = d.Get("python").(map[string]interface{})
	c.True(m["python"].(bool))
}

func TestCache(t *testing.T) {
	c := check.New(t)

	cPrev := 0.0
	ncPrev := 0.0

	cacheme := check.Fixture("cache/cacheme.py")
	neverCache := check.Fixture("cache/never_cache.py")

	for i := 0; i < 5; i++ {
		d := New(config.New().InDir(check.Fixture("cache")))
		err := d.LoadData(cacheme)
		c.MustNotError(err)

		err = d.LoadData(neverCache)
		c.MustNotError(err)

		cts := d.Get("cacheme").(map[string]interface{})["ts"].(float64)

		if i%2 == 0 {
			c.NotEqual(cts, cPrev)
		} else {
			c.Equal(cts, cPrev)
			os.Chtimes(cacheme, time.Now(), time.Now())
		}

		cPrev = cts

		ncts := d.Get("never_cache").(map[string]interface{})["ts"].(float64)
		c.NotEqual(ncts, ncPrev)
		ncPrev = ncts
	}
}

func TestLoadDir(t *testing.T) {
	c := check.New(t)
	d := New(config.New())

	dir := check.Fixture("just/a/dir")
	err := os.MkdirAll(dir, 0750)
	c.MustNotError(err)

	err = d.LoadData(dir)
	c.NotError(err)
	c.Len(d.ds, 0)
}

func TestErrors(t *testing.T) {
	c := check.New(t)
	d := New(config.New())

	err := d.LoadData(check.Fixture("blerp.json"))
	c.Error(err)
}

func TestBinError(t *testing.T) {
	c := check.New(t)
	d := New(config.New())

	err := d.LoadData(check.Fixture("fail.py"))
	c.Error(err)
}

func TestJSONError(t *testing.T) {
	c := check.New(t)
	d := New(config.New())

	err := d.LoadData(check.Fixture("invalid.json"))
	c.Error(err)
}
