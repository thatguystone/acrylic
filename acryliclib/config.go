package acryliclib

import (
	"os"
	"runtime"
)

// Config controls all aspects of how the site is built.
type Config struct {
	Root  string // Where the site files live, relative to current directory
	Theme string // Name of the theme to use

	// Date format to use when printing dates
	DateFormat string `yaml:"dateFormat"`

	FileMode   os.FileMode // For generated content. Defaults to 0640.
	DataDir    string      `yaml:"dataDir"`    // Defaults to "data"
	ContentDir string      `yaml:"contentDir"` // Defaults to "content"
	LayoutsDir string      `yaml:"layoutsDir"` // Defaults to "layouts"
	PublicDir  string      `yaml:"publicDir"`  // Defaults to "public"
	ThemesDir  string      `yaml:"themesDir"`  // Defaults to "themes"

	// If generated HTML should be minified
	MinifyHTML bool `yaml:"minifyHTML"`

	// If coffee/dart/etc should be rendered to JS
	RenderJS bool `yaml:"renderJS"`

	// If all found js files should be combined. Only used if RenderJS == true
	SingleJS bool `yaml:"singleJS"`

	// How to minify rendered JS
	MinifyJS Minifier `yaml:"minifyJS"`

	// If less/sass/etc should be rendered to CSS
	RenderCSS bool `yaml:"renderCSS"`

	// If all found css files should be combined. Only used if RenderCSS == true
	SingleCSS bool `yaml:"singleCSS"`

	// How to minify rendered CSS
	MinifyCSS Minifier `yaml:"minifyCSS"`

	// If your assets are sensitive to ordering (one js file must be loaded
	// before another), it's necessary to check that the unified asset files
	// have assets in the correct order. If they're not sensitive to ordering,
	// set this to True.
	//
	// It's possible to have irreconcilable asset ordering if you have
	// something like the following:
	//
	//  page1:
	//    {% js "one.js" %}
	//    {% js "two.js" %}
	//
	//  page2:
	//    {% js "two.js" %}
	//    {% js "one.js" %}
	//
	// The inversion of include order here is impossible to maintain in a
	// combined file. This option forces verification that everything is as it
	// should be, or it causes the build to fail.
	//
	// This check is only run when SingleJS/CSS == true.
	UnorderedJS  bool
	UnorderedCSS bool

	Jobs uint // How many jobs may be run in parallel. Defaults to GOMAXPROCS*2.
}

func (cfg *Config) setDefaults() {
	if cfg.Root == "" {
		cfg.Root = "."
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}

	if cfg.DateFormat == "" {
		cfg.DateFormat = sDateFormat
	}

	if cfg.ContentDir == "" {
		cfg.ContentDir = "content"
	}

	if cfg.LayoutsDir == "" {
		cfg.LayoutsDir = "layouts"
	}

	if cfg.PublicDir == "" {
		cfg.PublicDir = "public"
	}

	if cfg.ThemesDir == "" {
		cfg.ThemesDir = "themes"
	}

	if cfg.FileMode == 0 {
		cfg.FileMode = 0640
	}

	if cfg.Jobs == 0 {
		cfg.Jobs = uint(runtime.GOMAXPROCS(-1)) * 2
	}
}
