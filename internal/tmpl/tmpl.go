package tmpl

import (
	"github.com/flosch/pongo2"
	"github.com/thatguystone/acrylic/internal/config"
	"github.com/thatguystone/acrylic/internal/data"
	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/imgs"
	"github.com/thatguystone/acrylic/internal/pages"
	"github.com/thatguystone/cog/clog"
)

MAKE ALL URLS RELATIVE: ABSOLUTE URLS ARE STUPID

// T wraps all templating operations
type T struct {
	args    Args
	tmplSet *pongo2.TemplateSet
}

// A Tmpl is a compiled template
type Tmpl struct {
	args Args
	tmpl *pongo2.Template
}

// Args are things that are passed around everywhere
type Args struct {
	Cfg   *config.C
	Log   *clog.Logger
	Imgs  *imgs.Imgs
	Data  *data.D
	Pages *pages.Ps
}

func NewT(args Args) *T {
	return &T{
		args: args,
		tmplSet: pongo2.NewSet(
			"acrylic",
			pongo2.MustNewLocalFileSystemLoader(args.Cfg.TemplatesDir)),
	}
}

func (t *T) Compile(c string) (Tmpl, error) {
	tmpl, err := t.tmplSet.FromString(c)
	return Tmpl{
		args: t.args,
		tmpl: tmpl,
	}, err
}

func (t Tmpl) Render(f file.F, extraVars pongo2.Context) (string, error) {
	ctx := pongo2.Context{
		"ac": ac{
			args: t.args,
			f:    f,
		},
	}
	ctx = ctx.Update(extraVars)

	return t.tmpl.Execute(ctx)
}
