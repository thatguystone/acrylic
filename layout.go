package toner

import (
	"bytes"
	"io"

	p2 "github.com/flosch/pongo2"
)

type layout struct {
	tpl      *p2.Template
	s        *site
	which    string
	filePath string
	content  string // Used for internal templates
}

func (lo *layout) execute(ctx p2.Context, pc []byte, fw io.Writer) error {
	// NEED TO PROVIDE LAYOUT relPath SO IT DOESN'T USE CONTENT'S; ALSO NEED FILTER `contentRel` TO GET PATHS SPECIFIED BY CONTENT (IE. HEADER IMG FOR BLOG POSTS)
	ctx.Update(p2.Context{
		"Content": p2.AsSafeValue(string(pc)),
	})

	var b *bytes.Buffer
	w := fw

	if lo.s.cfg.MinifyHTML {
		b = &bytes.Buffer{}
		w = b
	}

	err := lo.tpl.ExecuteWriter(ctx, w)
	if err != nil {
		return err
	}

	if lo.s.cfg.MinifyHTML {
		err = lo.s.min.Minify("text/html", fw, bytes.NewReader(b.Bytes()))
	}

	return err
}
