package toner

import "os"

type Config struct {
	Root  string // Where the site files live, relative to current directory
	Theme string // Name of the theme to use

	DataDir    string // Defaults to "data"
	ContentDir string // Defaults to "content"
	LayoutsDir string // Defaults to "layouts"
	PublicDir  string // Defaults to "public"
	ThemesDir  string // Defaults to "themes"

	MinifyHTML bool        // If generated HTML should be minified
	FileMode   os.FileMode // Defaults to 0640

	Jobs uint // How many jobs may be run in parallel. Defaults to GOMAXPROCS.
}

func (cfg *Config) reload() error {
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
		cfg.Jobs = 1 //uint(runtime.GOMAXPROCS(-1))
	}

	return nil
}
