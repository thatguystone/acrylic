package toner

import (
	"fmt"
	"path/filepath"
	"strings"

	p2 "github.com/flosch/pongo2"
)

func init() {
	p2.RegisterTag("img", imgTag)
	p2.RegisterTag("img_src", imgSrcTag)
}

type imgTagNodeBase struct {
	p2RelNode
	src  p2.IEvaluator
	ext  p2.IEvaluator
	w    p2.IEvaluator
	h    p2.IEvaluator
	crop p2.IEvaluator
}

type imgTagNode struct {
	imgTagNodeBase
}

type imgSrcTagNode struct {
	imgTagNodeBase
}

func (n imgTagNodeBase) getImg(ctx *p2.ExecutionContext) (img img, err *p2.Error) {
	evals := [...]p2.IEvaluator{
		n.src,
		n.ext,
		n.w,
		n.h,
		n.crop,
	}

	vals := [len(evals)]*p2.Value{}

	for i, ev := range evals {
		if ev == nil {
			continue
		}

		var v *p2.Value
		v, err = ev.Evaluate(ctx)
		if err != nil {
			return
		}

		vals[i] = v
	}

	i := 0

	if !vals[i].IsString() {
		err = ctx.Error("img: src must be a string", nil)
		return
	}

	img.src = vals[i].String()

	i++
	if vals[i] != nil {
		if !vals[i].IsString() {
			err = ctx.Error("img: extension must be a string", nil)
			return
		}

		ext := vals[i].String()

		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}

		okExt := false
		for _, e := range imgExts {
			if ext == e {
				okExt = true
				break
			}
		}

		if !okExt {
			err = ctx.Error(fmt.Sprintf(
				"img: %s is an invalid image extension, must be one of %v",
				ext,
				imgExts),
				nil)
			return
		}

		img.ext = ext
	} else {
		img.ext = filepath.Ext(img.src)
	}

	i++
	if vals[i] != nil {
		if !vals[i].IsInteger() {
			err = ctx.Error("img: width must be an integer", nil)
			return
		}

		w := vals[i].Integer()
		if w < 0 {
			err = ctx.Error("img: width must be greather than 0", nil)
			return
		}

		img.w = w
	}

	i++
	if vals[i] != nil {
		if !vals[i].IsInteger() {
			err = ctx.Error("img: height must be an integer", nil)
			return
		}

		h := vals[i].Integer()
		if h < 0 {
			err = ctx.Error("img: height must be greather than 0", nil)
			return
		}

		img.h = h
	}

	i++
	if vals[i] != nil {
		if !vals[i].IsString() {
			err = ctx.Error("img: crop must be a string", nil)
			return
		}

		switch vals[i].String() {
		case "none":
			img.crop = cropNone

		case "centered", "center":
			img.crop = cropCentered

		case "left":
			img.crop = cropLeft

		default:
			err = ctx.Error(
				fmt.Sprintf("img: unrecognized crop argument: %s", vals[2].String()),
				nil)
			return
		}
	}

	return
}

func (n imgSrcTagNode) Execute(
	ctx *p2.ExecutionContext,
	w p2.TemplateWriter) *p2.Error {

	s := ctx.Public[privSiteKey].(*site)
	c := ctx.Public[contentKey].(*content)
	currFile := n.contentRel(c)

	if parentRel, ok := ctx.Public[parentRelKey]; ok {
		currFile = parentRel.(string)
	}

	img, perr := n.getImg(ctx)
	if perr != nil {
		return perr
	}

	ic, err := s.findContent(currFile, img.src)
	if err != nil {
		s.errs.add(currFile, fmt.Errorf("img: file not found: %v", err))
		return nil
	}

	cgi, ok := ic.gen.(contentGenImg)
	if !ok {
		s.errs.add(currFile,
			fmt.Errorf("img: %s is not an image, have %s",
				img.src,
				ic.gen.(contentGener).humanName()))
		return nil
	}

	imgW, imgH, path, err := cgi.scale(img)
	if err != nil {
		s.errs.add(currFile, fmt.Errorf("img: failed to scale %s: %v", img.src, err))
		return nil
	}

	ctx.Public["w"] = imgW
	ctx.Public["h"] = imgH

	_, err = w.WriteString(c.relDest(path))
	if err != nil {
		return ctx.Error(err.Error(), nil)
	}

	return nil
}

func (n imgTagNode) Execute(
	ctx *p2.ExecutionContext,
	w p2.TemplateWriter) *p2.Error {

	s := ctx.Public[privSiteKey].(*site)
	c := ctx.Public[contentKey].(*content)

	img, perr := n.getImg(ctx)
	if perr != nil {
		return perr
	}

	imgctx := p2.Context{}
	imgctx.Update(ctx.Public)
	imgctx.Update(p2.Context{
		parentRelKey: n.contentRel(c),
		"src":        img.src,
		"ext":        img.ext,
		"w":          img.w,
		"h":          img.h,
		"crop":       img.crop.String(),
	})

	lo := s.findLayout(c.cpath, "_img")
	err := lo.execute(imgctx, w)
	if err != nil {
		return ctx.Error(fmt.Sprintf("img: failed to write tag: %s", err), nil)
	}

	return nil
}

func imgParseArgs(d *p2.Parser, s *p2.Token, args *p2.Parser) (base imgTagNodeBase, err *p2.Error) {
	if args.Count() == 0 {
		err = args.Error("img: an image path is required", nil)
		return
	}

	src, err := args.ParseExpression()
	if err != nil {
		return
	}

	base.p2RelNode = p2RelFromToken(s)
	base.src = src

	for args.Remaining() > 0 {
		t := args.MatchType(p2.TokenIdentifier)
		if t == nil {
			err = args.Error(
				fmt.Sprintf("img: unexpected token: expected %s, got %d",
					p2.TokenIdentifier,
					args.Current().Typ),
				t)
			return
		}

		var where *p2.IEvaluator

		switch t.Val {
		case "ext":
			where = &base.ext

		case "crop":
			where = &base.crop

		case "height":
			where = &base.h

		case "width":
			where = &base.w
		}

		if where == nil {
			err = args.Error(
				fmt.Sprintf("img: unrecognized option: %s", t.Val),
				t)
			return
		}

		arg := args.Match(p2.TokenSymbol, "=")
		if arg == nil {
			err = args.Error(
				fmt.Sprintf("img: %s requires an argument", t.Val),
				t)
			return
		}

		*where, err = args.ParseExpression()
		if err != nil {
			return
		}
	}

	return
}

func imgSrcTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	base, err := imgParseArgs(d, s, args)
	if err != nil {
		return nil, err
	}

	return imgSrcTagNode{base}, nil
}

func imgTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
	base, err := imgParseArgs(d, s, args)
	if err != nil {
		return nil, err
	}

	return imgTagNode{base}, nil
}
