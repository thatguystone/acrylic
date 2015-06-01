package toner

import (
	"os"
	"runtime"
)

type Config struct {
	Root  string // Where the site files live, relative to current directory
	Theme string // Name of the theme to use

	FileMode   os.FileMode // For generated content. Defaults to 0640.
	DataDir    string      // Defaults to "data"
	ContentDir string      // Defaults to "content"
	LayoutsDir string      // Defaults to "layouts"
	PublicDir  string      // Defaults to "public"
	ThemesDir  string      // Defaults to "themes"

	MinifyHTML bool     // If generated HTML should be minified
	RenderJS   bool     // If coffee/dart/etc should be rendered to JS
	SingleJS   bool     // If all found js files should be combined. Only used if RenderJS == true
	MinifyJS   Minifier // How to minify rendered JS
	RenderCSS  bool     // If less/sass/etc should be rendered to CSS
	SingleCSS  bool     // If all found css files should be combined. Only used if RenderCSS == true
	MinifyCSS  Minifier // How to minify rendered CSS

	Jobs uint // How many jobs may be run in parallel. Defaults to GOMAXPROCS.
}

func (cfg *Config) setDefaults() {
	if cfg.Root == "" {
		cfg.Root = "."
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "data"
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
		cfg.Jobs = uint(runtime.GOMAXPROCS(-1))
	}
}
