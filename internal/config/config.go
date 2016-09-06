package config

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// C stands for "config".
type C struct {
	// Site title
	Title string

	// URL to use for absolute URLs
	// TODO(astone): should this exist?
	SiteURL string

	// Directories where to find stuff
	AssetsDir    string
	CacheDir     string
	ContentDir   string
	DataDir      string
	PublicDir    string
	TemplatesDir string

	// Number of posts to put per page
	PerPage int

	// CSS/SCSS files to throw on the pages. These paths are relative to
	// AssetsDir.
	CSS []string

	// JS files to throw on the pages. These paths are relative to AssetsDir.
	JS []string

	// Args to run a sass compile
	SassCompiler []string

	// Args to run a JS compiler
	JSCompiler []string

	// If assets should be cache-busted
	CacheBust bool

	// For debugging
	Debug     bool
	DebugAddr string
}

var sassCmds = [][]string{
	[]string{"sass", "--scss"},
	[]string{"sassc"},
}

func New() *C {
	sassCmd := sassCmds[0]
	for _, cmd := range sassCmds {
		_, err := exec.LookPath(cmd[0])
		if err == nil {
			sassCmd = cmd
			break
		}
	}

	return &C{
		AssetsDir:    "assets/",
		CacheDir:     ".cache/",
		ContentDir:   "content/",
		DataDir:      "data/",
		PublicDir:    "public/",
		TemplatesDir: "templates/",
		PerPage:      5,
		SassCompiler: sassCmd,
		CacheBust:    true,
		DebugAddr:    ":8000",
	}
}

// Load extra configs on top of this config.
func (c *C) Load(files ...string) error {
	for _, file := range files {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read config file: %v", err)
		}

		err = yaml.Unmarshal(b, c)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config file %s: %v", file, err)
		}
	}

	return nil
}

// InDir prefixes each non-absolute path in C with the given dir.
func (c C) InDir(dir string) *C {
	pfx := []*string{
		&c.AssetsDir,
		&c.CacheDir,
		&c.ContentDir,
		&c.DataDir,
		&c.PublicDir,
		&c.TemplatesDir,
	}

	for _, p := range pfx {
		if !path.IsAbs(*p) {
			*p = filepath.Join(dir, *p)
		}
	}

	return &c
}
