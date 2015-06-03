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

func (lo *layout) execute(ctx p2.Context, fw io.Writer) error {
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
