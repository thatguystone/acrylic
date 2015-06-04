package toner

type Minifier interface {
	Minify(path string) error
}
