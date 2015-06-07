package acrylib

type contentGenBlob struct {
}

func getBlobGener(s *site, c *content, ext string) (contentGener, contentType) {
	return &contentGenBlob{}, contBlob
}

func (gb *contentGenBlob) claimDest(c *content) (dstPath string, alreadyClaimed bool, err error) {
	dstPath, alreadyClaimed, err = c.claimDest("")
	return
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
