package acryliclib

// Minifier provides asset minification.
type Minifier interface {
	Minify(path string) error
}
