package acryliclib

type Minifier interface {
	Minify(path string) error
}
