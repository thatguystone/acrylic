package acryliclib

import (
	"fmt"
	"io"
)

type contentGenGetter func(s *site, c *content, ext string) (contentGener, contentType)

type contentGener interface {
	// Claim the file you're going to output
	claimDest(c *content) (dstPath string, alreadyClaimed bool, err error)

	// Render the content itself, not the full page
	render(s *site, c *content) (content []byte, err error)

	// Generate the full page, optionally writing the file yourself
	generate(content []byte, dstPath string, s *site, c *content) (wroteOwnFile bool, err error)
}

type contentGenWrapper struct {
	s            *site
	c            *content
	gener        contentGener
	contType     contentType
	content      string
	contentGened chan struct{}
}

type contentType int

const (
	contInvalid contentType = iota
	contPage
	contJS
	contCSS
	contImg
	contBlob
)

var genGetters = []contentGenGetter{
	getContentPageGener,
	getContentJSGener,
	getContentCSSGener,
	getContentImgGener,
	getBlobGener,
}

func getContentGener(s *site, c *content, ext string) contentGenWrapper {
	for _, gg := range genGetters {
		contentGener, contType := gg(s, c, ext)
		if contentGener != nil {
			return contentGenWrapper{
				s:            s,
				c:            c,
				gener:        contentGener,
				contType:     contType,
				contentGened: make(chan struct{}),
			}
		}
	}

	panic(fmt.Errorf("no content generator found for %s", ext))
}

// An interface, ready for type hurling
func (gw *contentGenWrapper) getGener() interface{} {
	return gw.gener
}

func (gw *contentGenWrapper) generatePage() (dstPath string) {
	dstPath, alreadyClaimed, err := gw.gener.claimDest(gw.c)
	if err != nil {
		close(gw.contentGened)
		gw.s.errs.add(gw.c.f.srcPath, err)
		return
	}

	if alreadyClaimed {
		return
	}

	content, err := gw.gener.render(gw.s, gw.c)

	gw.content = string(content)
	if len(gw.content) == 0 {
		// For recursive rendering: don't allow getContent() to deadlock
		gw.content = " "
	}

	close(gw.contentGened)

	if err == nil {
		wroteOwnFile := false
		wroteOwnFile, err = gw.gener.generate(content, dstPath, gw.s, gw.c)

		if err == nil && !wroteOwnFile {
			err = gw.writeFile(dstPath, content)
		}
	}

	if err != nil {
		gw.s.errs.add(gw.c.f.srcPath, err)
		return
	}

	return
}

// Layouts call into this from `generatePage()` -> `Page.Content` in a
// template (which produces a call to layoutPageCtx.Content()) -> here, so
// make sure that gw.content is set to something before getting here, or
// there's going to be some deadlock.
func (gw *contentGenWrapper) getContent() string {
	if len(gw.content) == 0 {
		gw.generatePage()
		<-gw.contentGened
	}

	return gw.content
}

func (gw *contentGenWrapper) is(contType contentType) bool {
	return gw.contType == contType
}

func (gw *contentGenWrapper) humanName() string {
	return gw.contType.String()
}

func (gw *contentGenWrapper) writeFile(dstPath string, content []byte) (err error) {
	f, err := gw.s.fCreate(dstPath)
	if err != nil {
		return
	}

	defer f.Close()

	w, err := f.Write(content)
	if err != nil {
		return
	}

	if w != len(content) {
		err = io.ErrShortWrite
		return
	}

	err = f.Close()
	return
}

func (contType contentType) String() string {
	switch contType {
	case contPage:
		return "page"
	case contJS:
		return "js"
	case contCSS:
		return "css"
	case contImg:
		return "image"
	case contBlob:
		return "binary blob"
	}

	panic(fmt.Errorf("unrecognized content type: %d", contType))
}
