package acrylib

import "path/filepath"

type contentGenBlob struct {
}

func getBlobGener(s *site, c *content, ext string) (contentGener, contentType) {
	return &contentGenBlob{}, contBlob
}

func (gb *contentGenBlob) finalExt(c *content) string {
	return filepath.Ext(c.f.srcPath)
}

func (gb *contentGenBlob) render(s *site, c *content) (content []byte, err error) {
	return
}

func (gb *contentGenBlob) generate(content []byte, dstPath string, s *site, c *content) (
	wroteOwnFile bool,
	err error) {

	s.stats.addBlob()

	wroteOwnFile = true

	return
}
