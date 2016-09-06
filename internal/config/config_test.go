package config

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestLoadErrors(t *testing.T) {
	c := check.New(t)

	cfg := New()
	err := cfg.Load(check.Fixture("narp.yml"))
	c.Error(err)

	err = cfg.Load(check.Fixture("invalid.yml"))
	c.Error(err)
}

func TestInDir(t *testing.T) {
	c := check.New(t)

	cfg := New()
	err := cfg.Load(check.Fixture("one.yml"), check.Fixture("two.yml"))
	c.MustNotError(err)
	c.Log(cfg)

	cfg.JS = []string{
		"js/test.js",
	}

	cfg.CSS = []string{
		"js/test.css",
	}

	cfg = cfg.InDir("blah/")
	c.Equal(cfg.AssetsDir, "blah/changed/assets")
	c.Equal(cfg.CacheDir, "blah/changed/cache")
	c.Equal(cfg.DataDir, "blah/data")
}
