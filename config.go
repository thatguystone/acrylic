package main

type config struct {
	// Directories where to find stuff
	AssetsDir    string
	CacheDir     string
	ContentDir   string
	DataDir      string
	PublicDir    string
	TemplatesDir string

	// Site title
	Title string

	// Items to put in the nav bar
	Navs nav

	// Number of posts to put per page
	PerPage int

	// CSS/SASS files to throw on the pages
	CSS []string

	// JS files to throw on the pages
	JS []string

	// Args for JS compiler
	JSCompiler []string

	// For debugging
	Debug     bool
	DebugPort int
}

type nav struct {
	// Title of the page
	Title string

	// Link to the page
	URL string
}

func newConfig() *config {
	return &config{
		AssetsDir:    "assets/",
		CacheDir:     "cache/",
		ContentDir:   "content/",
		DataDir:      "data/",
		PublicDir:    "public/",
		TemplatesDir: "templates/",
		PerPage:      5,
		DebugPort:    8000,
	}
}
