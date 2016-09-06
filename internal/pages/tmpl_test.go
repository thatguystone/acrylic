package pages

import (
	"fmt"
	"io/ioutil"

	"github.com/flosch/pongo2"
	"github.com/thatguystone/acrylic/internal/file"
)

type tmplCompiler struct{}
type tmplRenderer struct{}
type tmplErrCompiler struct{}

func (tmplCompiler) Compile(s string) (TmplRenderer, error) {
	return tmplRenderer{}, nil
}

func (tmplRenderer) Render(f file.F, extraVars pongo2.Context) (string, error) {
	c, err := ioutil.ReadFile(f.Src)
	return string(c), err
}

func (tmplErrCompiler) Compile(s string) (TmplRenderer, error) {
	return nil, fmt.Errorf("compile failed")
}
