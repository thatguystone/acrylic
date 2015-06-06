package acryliclib

type renderer interface {
	renders(ext string) bool
	alwaysRender() bool
	render(b []byte) ([]byte, error)
}

func findRenderer(ext string, rends []renderer) renderer {
	for _, r := range rends {
		if r.renders(ext) {
			return r
		}
	}

	return nil
}
