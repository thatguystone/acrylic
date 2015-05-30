package toner

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"

	p2 "github.com/flosch/pongo2"
)

type layout struct {
	tpl     *p2.Template
	tpla    tplAssets
	path    string
	content string // Only set for default templates during loading
}

type loPage struct {
	Content *p2.Value
}

func (lo *layout) preexecute(s *site) error {
	ctx := p2.Context{
		assetsKey:  &lo.tpla,
		relPathKey: filepath.Dir(lo.path),
	}

	err := lo.execute(s, ctx, nil, ioutil.Discard)
	lo.tpla.setRendered()

	return err
}

func (lo *layout) execute(s *site, ctx p2.Context, pc []byte, fw io.Writer) error {
	ctx.Update(p2.Context{
		"Page": loPage{
			Content: p2.AsSafeValue(string(pc)),
		},
	})

	var b *bytes.Buffer
	w := fw

	minify := s.cfg.MinifyHTML && lo.tpla.rendered

	if minify {
		b = &bytes.Buffer{}
		w = b
	}

	err := lo.tpl.ExecuteWriter(ctx, w)
	if err != nil {
		return err
	}

	if minify {
		err = s.min.Minify("text/html", fw, bytes.NewReader(b.Bytes()))
	}

	return err
}
