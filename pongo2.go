package toner

import (
	"fmt"
	"io"
	"reflect"

	p2 "github.com/flosch/pongo2"
)

const (
	contentKey   = "__tonerContent__"
	parentRelKey = "__tonerParentRel__"
	privSiteKey  = "__tonerSite__"
)

func init() {
	p2.RegisterTag("js", jsTag)
	p2.RegisterTag("css", cssTag)

	p2.RegisterTag("js_all", jsAllTag)
	p2.RegisterTag("css_all", cssAllTag)

	// NEED TO PROVIDE LAYOUT relPath SO IT DOESN'T USE CONTENT'S; ALSO NEED FILTER `contentRel` TO GET PATHS SPECIFIED BY CONTENT (IE. HEADER IMG FOR BLOG POSTS)
	// p2.RegisterFilter("content_rel", contentRelFilt)
}

type p2RelNode struct {
	file string
}

type assetTagNode struct {
	p2RelNode
	what    string
	srcs    []p2.IEvaluator
	genType reflect.Type
	writer  func(c *content, path string, w p2.TemplateWriter) error
}

type jsAllNode struct{}
type cssAllNode struct{}

type tagWriter interface {
	writeTag(path string, w io.Writer) error
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
			err = n.writer(c, relPath, w)
		}

		if err != nil {
			s.errs.add(currFile, fmt.Errorf("%s: %v", n.what, err))
		}
	}

	return nil
}

func jsTagWriter(c *content, path string, w p2.TemplateWriter) error {
	return c.gen.(contentGenJS).rend.(tagWriter).writeTag(path, w)
}

func cssTagWriter(c *content, path string, w p2.TemplateWriter) error {
	return c.gen.(contentGenCSS).rend.(tagWriter).writeTag(path, w)
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

func assetTag(what string, d *p2.Parser, s *p2.Token, args *p2.Parser) (assetTagNode, *p2.Error) {
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
	n.writer = jsTagWriter

	return n, nil
}

func cssTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	n, err := assetTag("css", d, s, args)
	if err != nil {
		return nil, err
	}

	n.genType = reflect.TypeOf(contentGenCSS{})
	n.writer = cssTagWriter

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
