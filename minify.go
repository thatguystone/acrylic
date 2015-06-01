package toner

type Minifier interface {
	Minify(dstPath string, c []byte) ([]byte, error)
}
