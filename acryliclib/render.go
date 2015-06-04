package acryliclib

type renderer interface {
	renders(ext string) bool
	alwaysRender() bool
	render(b []byte) ([]byte, error)
}
