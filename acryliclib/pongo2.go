package acryliclib

import (
	"bytes"
	"fmt"
	"reflect"

	p2 "github.com/flosch/pongo2"
)

const (
	contentKey   = "__acrylicContent__"
	parentRelKey = "__acrylicParentRel__"
	privSiteKey  = "__acrylicSite__"
)

func init() {
	p2.RegisterTag("content", contentTag)

	p2.RegisterTag("js", jsTag)
	p2.RegisterTag("css", cssTag)

	p2.RegisterTag("js_all", jsAllTag)
	p2.RegisterTag("css_all", cssAllTag)

	// TODO(astone): `contentRel` to get paths specified by content (ie. header img for blog posts)
	// p2.RegisterFilter("content_rel", contentRelFilt)
}

type p2RelNode struct {
	file string
}

type contentNode struct{}

type assetTagNode struct {
	p2RelNode
	what    string
	srcs    []p2.IEvaluator
	genType reflect.Type
}

type jsAllNode struct{}
type cssAllNode struct{}

func (n contentNode) Execute(
	ctx *p2.ExecutionContext,
	w p2.TemplateWriter) *p2.Error {

	s := ctx.Public[privSiteKey].(*site)
	c := ctx.Public[contentKey].(*content)

	c.kickAssets = true

	b := bytes.Buffer{}
	err := c.templatize(&b)
	if err != nil {
		s.errs.add(c.f.srcPath,
			fmt.Errorf("content: failed to templatize: %v", err))
		return nil
	}

	rc, err := c.gen.(contentGenPage).rend.render(b.Bytes())
	if err != nil {
		s.errs.add(c.f.srcPath,
			fmt.Errorf("content: failed to render: %v", err))
		return nil
	}

	_, err = w.Write(rc)
	if err != nil {
		s.errs.add(c.f.srcPath,
			fmt.Errorf("content: failed to write: %v", err))
		return nil
	}

	return nil
}

func (n assetTagNode) Execute(
	ctx *p2.ExecutionContext,
	w p2.TemplateWriter) *p2.Error {

	s := ctx.Public[privSiteKey].(*site)
	pc := ctx.Public[contentKey].(*content)
	currFile := n.contentRel(pc)

	for _, src := range n.srcs {
		v, perr := src.Evaluate(ctx)
		if perr != nil {
			return perr
		}

		path := v.String()

		c, err := s.findContent(currFile, path)
		if err != nil {
			s.errs.add(currFile, fmt.Errorf("%s: file not found: %s", n.what, err))
			continue
		}

		ok := reflect.TypeOf(c.gen) == n.genType
		if !ok {
			s.errs.add(currFile,
				fmt.Errorf("%s: `%s` is not a %s file, have %s",
					n.what, path, n.what,
					c.gen.(contentGener).humanName()))
			continue
		}

		path, err = c.gen.(contentGener).generatePage()
		if err == nil {
			relPath := c.relDest(path)
			err = s.assets.writeTag(pc, path, relPath, w)
		}

		if err != nil {
			s.errs.add(currFile, fmt.Errorf("%s: %v", n.what, err))
		}
	}

	return nil
}

func (n jsAllNode) Execute(
	ctx *p2.ExecutionContext,
	w p2.TemplateWriter) *p2.Error {

	return nil
}

func (n cssAllNode) Execute(
	ctx *p2.ExecutionContext,
	w p2.TemplateWriter) *p2.Error {

	return nil
}

func tagParseExpressions(
	d *p2.Parser,
	s *p2.Token,
	args *p2.Parser) ([]p2.IEvaluator, *p2.Error) {

	var exps []p2.IEvaluator

	for args.Remaining() > 0 {
		exp, err := args.ParseExpression()
		if err != nil {
			return nil, err
		}

		exps = append(exps, exp)
	}

	return exps, nil
}

func contentTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	if args.Count() > 0 {
		return nil, args.Error("content: no arguments expected", nil)
	}

	return contentNode{}, nil
}

func assetTag(what string, d *p2.Parser, s *p2.Token, args *p2.Parser) (
	assetTagNode,
	*p2.Error) {

	srcs, err := tagParseExpressions(d, s, args)
	if err != nil {
		return assetTagNode{}, err
	}

	if len(srcs) == 0 {
		return assetTagNode{}, args.Error(
			fmt.Sprintf("%s: 1 or more arguments is required", what),
			nil)
	}

	n := assetTagNode{
		p2RelNode: p2RelFromToken(s),
		what:      what,
		srcs:      srcs,
	}

	return n, nil
}

func jsTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	n, err := assetTag("js", d, s, args)
	if err != nil {
		return nil, err
	}

	n.genType = reflect.TypeOf(contentGenJS{})

	return n, nil
}

func cssTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	n, err := assetTag("css", d, s, args)
	if err != nil {
		return nil, err
	}

	n.genType = reflect.TypeOf(contentGenCSS{})

	return n, nil
}

func jsAllTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	if args.Count() > 0 {
		return nil, args.Error("js_all: no arguments expected", nil)
	}

	return jsAllNode{}, nil
}

func cssAllTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	if args.Count() > 0 {
		return nil, args.Error("css_all: no arguments expected", nil)
	}

	return cssAllNode{}, nil
}

func p2RelFromToken(t *p2.Token) p2RelNode {
	f := t.Filename
	if f == "<string>" {
		f = ""
	}

	return p2RelNode{
		file: f,
	}
}

func (rn p2RelNode) contentRel(c *content) string {
	if rn.file != "" {
		return rn.file
	}

	return c.f.srcPath
}
