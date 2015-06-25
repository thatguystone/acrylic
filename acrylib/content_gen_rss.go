package acrylib

import "bytes"

type contentGenRSS struct {
}

func getRSSGener(s *site, c *content, ext string) (contentGener, contentType) {
	if ext != ".rss" {
		return nil, contInvalid
	}

	return &contentGenRSS{}, contRSS
}

func (gr *contentGenRSS) defaultGen() bool {
	return true
}

func (gr *contentGenRSS) finalExt(c *content) string {
	return ".rss"
}

func (gr *contentGenRSS) render(s *site, c *content) (content []byte, err error) {
	lo := s.findLayout(c.cpath, "_rss", true)

	b := &bytes.Buffer{}
	err = lo.execute(c.tplCont.forPage(), b)
	content = b.Bytes()

	return
}

func (gr *contentGenRSS) generate(
	content []byte,
	dstPath string,
	s *site,
	c *content) (wroteOwnFile bool, err error) {

	return
}
