package main

import "os/exec"

type config struct {
	// Site title
	Title string

	// URL to use for absolute URLs
	SiteURL string

	// Directories where to find stuff
	AssetsDir    string
	CacheDir     string
	ContentDir   string
	DataDir      string
	PublicDir    string
	TemplatesDir string

	// Items to put in the nav bar
	Nav []nav

	// Number of posts to put per page
	PerPage int

	// CSS/SCSS files to throw on the pages
	CSS []string

	// JS files to throw on the pages
	JS []string

	// Args to run a sass compile
	SassCompiler []string

	// Args to run a JS compiler
	JSCompiler []string

	// For debugging
	Debug     bool
	DebugAddr string
}

type nav struct {
	// Title of the page
	Title string

	// Link to the page
	URL string
}

var sassCmds = [][]string{
	[]string{"sass", "--scss"},
	[]string{"sassc"},
}

func newConfig() *config {
	sassCmd := sassCmds[0]
	for _, cmd := range sassCmds {
		_, err := exec.LookPath(cmd[0])
		if err == nil {
			sassCmd = cmd
			break
		}
	}

	return &config{
		AssetsDir:    "assets/",
		CacheDir:     ".cache/",
		ContentDir:   "content/",
		DataDir:      "data/",
		PublicDir:    "public/",
		TemplatesDir: "templates/",
		PerPage:      5,
		SassCompiler: sassCmd,
		DebugAddr:    ":8000",
	}
}
