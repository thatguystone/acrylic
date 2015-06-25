package acrylib

import (
	"fmt"
	"path/filepath"

	p2 "github.com/flosch/pongo2"
)

func init() {
	p2.RegisterTag("url", urlTag)

	p2.RegisterTag("js",
		func(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
			return assetTag("js", contJS, d, s, args)
		})
	p2.RegisterTag("css",
		func(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
			return assetTag("css", contCSS, d, s, args)
		})

	p2.RegisterTag("js_all",
		func(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
			return assetAllTag("js", contJS, renderJS{}, d, s, args)
		})

	p2.RegisterTag("css_all",
		func(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
			return assetAllTag("css", contCSS, renderCSS{}, d, s, args)
		})

	// These are, awkwardly enough, here since init() functions don't have an order
	for _, t := range bannedContentTags {
		p2ContentSet.BanTag(t)
	}

	for _, f := range bannedContentFilters {
		p2ContentSet.BanFilter(f)
	}
}

type p2RelNode struct {
	file string
}

type urlNode struct {
	p2RelNode
	exp p2.IEvaluator
}

type assetTagNode struct {
	p2RelNode
	what     string
	contType contentType
	srcs     []p2.IEvaluator
}

type assetAllNode struct {
	p2RelNode
	what     string
	contType contentType
	tagger   tagWriter
}

func (n urlNode) Execute(ctx *p2.ExecutionContext, w p2.TemplateWriter) *p2.Error {
	s := ctx.Public[privSiteKey].(*site)
	c := ctx.Public[contentKey].(*content)

	v, perr := n.exp.Evaluate(ctx)
	if perr != nil {
		return perr
	}

	currFile := n.contentRel(c)
	fc, err := s.findContent(c, currFile, v.String())
	if err != nil {
		s.errs.add(c.f.srcPath, fmt.Errorf("url: file not found: %s", err))
		return nil
	}

	path := fc.gen.generatePage()
	relPath := c.relDest(path)
	_, err = w.WriteString(relPath)

	if err != nil {
		s.errs.add(c.f.srcPath, fmt.Errorf("url: %v", err))
	}

	return nil
}

func (n assetTagNode) Execute(ctx *p2.ExecutionContext, w p2.TemplateWriter) *p2.Error {
	s := ctx.Public[privSiteKey].(*site)
	c := ctx.Public[contentKey].(*content)
	assetOrd := ctx.Public[assetOrdKey].(*assetOrdering)

	currFile := n.contentRel(c)

	for _, src := range n.srcs {
		v, perr := src.Evaluate(ctx)
		if perr != nil {
			return perr
		}

		path := v.String()
		relPath := path

		if !isRemoteURL(path) {
			relContent, err := s.findContent(c, currFile, path)
			if err != nil {
				s.errs.add(c.f.srcPath,
					fmt.Errorf("%s: file not found: %s", n.what, err))
				continue
			}

			if !relContent.gen.is(n.contType) {
				s.errs.add(c.f.srcPath,
					fmt.Errorf("%s: `%s` is not a %s file, have %s",
						n.what, path, n.what,
						relContent.gen.humanName()))
				continue
			}

			path = relContent.gen.generatePage()
			relPath = c.relDest(path)
		}

		err := s.assets.addToOrderingAndWrite(assetOrd, path, relPath, w)

		if err != nil {
			s.errs.add(c.f.srcPath, fmt.Errorf("%s: %v", n.what, err))
		}
	}

	return nil
}

func (n assetAllNode) Execute(ctx *p2.ExecutionContext, w p2.TemplateWriter) *p2.Error {
	s := ctx.Public[privSiteKey].(*site)
	c := ctx.Public[contentKey].(*content)

	if !s.assets.getType(n.contType).doCombine {
		return nil
	}

	relPath := c.relDest(filepath.Join(s.cfg.Root, combinedName))
	relPath += "." + n.what

	_, err := n.tagger.writeTag(relPath, w)
	if err != nil {
		s.errs.add(c.f.srcPath, fmt.Errorf("%s_all: %v", n.what, err))
	}

	return nil
}

func p2TagParseExpressions(
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

func urlTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	exp, err := args.ParseExpression()
	if err != nil {
		return nil, err
	}

	if args.Remaining() != 0 {
		return nil, args.Error("url: only 1 argument expected", nil)
	}

	n := urlNode{
		p2RelNode: p2RelFromToken(s),
		exp:       exp,
	}

	return n, nil
}

func assetTag(
	what string,
	contType contentType,
	d *p2.Parser,
	s *p2.Token,
	args *p2.Parser) (assetTagNode, *p2.Error) {

	srcs, err := p2TagParseExpressions(d, s, args)
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
		contType:  contType,
	}

	return n, nil
}

func assetAllTag(
	what string,
	contType contentType,
	tagger tagWriter,
	d *p2.Parser,
	s *p2.Token,
	args *p2.Parser) (p2.INodeTag, *p2.Error) {

	if args.Count() > 0 {
		return nil, args.Error(fmt.Sprintf("%s_all: no arguments expected", what), nil)
	}

	n := assetAllNode{
		p2RelNode: p2RelFromToken(s),
		what:      what,
		contType:  contType,
		tagger:    tagger,
	}
	return n, nil
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
