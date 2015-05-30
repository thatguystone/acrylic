package toner

import (
	"fmt"

	p2 "github.com/flosch/pongo2"
)

func init() {
	p2.RegisterTag("js", jsTag)
	p2.RegisterTag("css", cssTag)

	p2.RegisterTag("js_tags", jsTags)
	p2.RegisterTag("css_tags", cssTags)
}

type jsCSSTagNode struct {
	js    bool
	paths []string
}

type jsCSSTagsNode bool

func getJSCSSString(js bool) string {
	if js {
		return "js"
	}
	return "css"
}

func (n jsCSSTagNode) Execute(
	ctx *p2.ExecutionContext,
	w p2.TemplateWriter) *p2.Error {

	tpla := ctx.Public[assetsKey].(*tplAssets)
	if tpla.rendered {
		return nil
	}

	relPath := ctx.Public[relPathKey].(string)

	for _, path := range n.paths {
		path = fRelPath(relPath, path)

		if n.js {
			tpla.addJS(path)
		} else {
			tpla.addCSS(path)
		}
	}

	return nil
}

func jsCSSTag(
	js bool,
	d *p2.Parser,
	s *p2.Token,
	args *p2.Parser) (p2.INodeTag, *p2.Error) {

	n := jsCSSTagNode{
		js:    js,
		paths: make([]string, args.Count()),
	}

	for i := range n.paths {
		tok := args.Get(i)

		// Only allow strings: values are evaluated once per template, so
		// allowing variables would result in possibly-stale values being
		// rendered into templates.
		if tok.Typ != p2.TokenString {
			return nil, args.Error(
				fmt.Sprintf("%s: arguments must be strings, not %s",
					getJSCSSString(js),
					tok),
				tok)
		}

		n.paths[i] = tok.Val
	}

	return n, nil
}

func jsTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	return jsCSSTag(true, d, s, args)
}

func cssTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	return jsCSSTag(false, d, s, args)
}

func (js jsCSSTagsNode) Execute(
	ctx *p2.ExecutionContext,
	w p2.TemplateWriter) *p2.Error {

	tpla := ctx.Public[assetsKey].(*tplAssets)
	relPath := ctx.Public[relPathKey].(string)

	var err error
	if js {
		err = tpla.writeJSTags(relPath, w)
	} else {
		err = tpla.writeCSSTags(relPath, w)
	}

	if err != nil {
		return ctx.Error(err.Error(), nil)
	}

	return nil
}

func jsCSSTags(
	js bool,
	d *p2.Parser,
	s *p2.Token,
	args *p2.Parser) (p2.INodeTag, *p2.Error) {

	if args.Count() > 0 {
		msg := fmt.Sprintf("Tag '%s_tags' accepts no arguments", getJSCSSString(js))
		return nil, args.Error(msg, nil)
	}

	return jsCSSTagsNode(js), nil
}

func jsTags(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	return jsCSSTags(true, d, s, args)
}

func cssTags(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	return jsCSSTags(false, d, s, args)
}
