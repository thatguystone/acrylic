package acrylib

import "os"

// Config controls all aspects of how the site is built.
type Config struct {
	Root  string // Where the site files live, relative to current directory
	Theme string // Name of the theme to use

	// Title to use for the site
	Title string

	// Base URL to use for public links
	URL string

	// Date format to use when printing dates
	DateFormat string `yaml:"dateFormat"`

	// At least how many words to include in content summaries. Summaries are
	// split on sentences, so a sentence that crosses this boundary will be
	// included in the summary.
	SummaryWords int

	// If content dated in the future should be generated and published
	PublishFuture bool

	// If content should be generated to .html files instead of
	// dir/index.html. For example, generate /content/about.html to
	// /public/about.html instead of /public/about/index.html.
	UglyURLs bool

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

	// Paths to various libraries that are needed by assets
	LessURL   string `yaml:"lessURL"`
	CoffeeURL string `yaml:"coffeeURL"`

	// If the build should be made reproducible (no time changes, etc)
	reproducibleBuild bool
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

	if cfg.SummaryWords <= 0 {
		cfg.SummaryWords = 70
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

	if cfg.LessURL == "" {
		cfg.LessURL = "//cdnjs.cloudflare.com/ajax/libs/less.js/2.5.1/less.min.js"
	}

	if cfg.CoffeeURL == "" {
		cfg.CoffeeURL = "//cdnjs.cloudflare.com/ajax/libs/coffee-script/1.9.3/coffee-script.min.js"
	}
}
