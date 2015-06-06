package acryliclib

import (
	"bytes"
	"fmt"
)

type contentGenPage struct {
	rend renderer
}

var contentPageRends = []renderer{
	renderMarkdown{},
	renderHTML{},
}

func getContentPageGener(s *site, c *content, ext string) (contentGener, contentType) {
	rend := findRenderer(ext, contentPageRends)
	if rend == nil {
		return nil, contInvalid
	}

	gp := &contentGenPage{
		rend: rend,
	}

	return gp, contPage
}

func (gp *contentGenPage) claimDest(c *content) (dstPath string, alreadyClaimed bool, err error) {
	dstPath, alreadyClaimed, err = c.claimDest(".html")
	return
}

func (gp *contentGenPage) render(s *site, c *content) (content []byte, err error) {
	b := bytes.Buffer{}
	err = c.templatize(&b)
	if err != nil {
		return
	}

	content, err = gp.rend.render(b.Bytes())
	if err != nil {
		err = fmt.Errorf("content: failed to render: %v", err)
		return
	}

	return
}

func (gp *contentGenPage) generate(content []byte, dstPath string, s *site, c *content) (
	wroteOwnFile bool,
	err error) {

	s.stats.addPage()
	wroteOwnFile = true

	f, err := s.fCreate(dstPath)
	if err != nil {
		err = fmt.Errorf("content: failed to create dest file: %v", err)
		return
	}

	defer f.Close()

	lo := s.findLayout(c.cpath, c.f.layoutName, true)
	assetOrd := assetOrdering{
		isPage: true,
	}

	err = lo.execute(c.loutCtx.forLayout(&assetOrd), f)
	if err != nil {
		err = fmt.Errorf("content: failed to render layout: %v", err)
		return
	}

	s.assets.addToOrderCheck(c.f.srcPath, assetOrd)

	return
}
