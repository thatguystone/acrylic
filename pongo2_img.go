package toner

// import (
// 	"fmt"

// 	p2 "github.com/flosch/pongo2"
// )

// func init() {
// 	p2.RegisterTag("img", imgTag)
// }

// type imgTagNode struct {
// 	src  p2.IEvaluator
// 	w    p2.IEvaluator
// 	h    p2.IEvaluator
// 	crop p2.IEvaluator
// }

// func (n imgTagNode) Execute(
// 	ctx *p2.ExecutionContext,
// 	w p2.TemplateWriter) *p2.Error {

// 	s := ctx.Public[siteKey].(*site)
// 	c := ctx.Public[contentKey].(content)

// 	evals := [...]p2.IEvaluator{
// 		n.src,
// 		n.w,
// 		n.h,
// 		n.crop,
// 	}

// 	vals := [len(evals)]*p2.Value{}

// 	for i, ev := range evals {
// 		if ev == nil {
// 			continue
// 		}

// 		v, err := n.src.Evaluate(ctx)
// 		if err != nil {
// 			return err
// 		}

// 		vals[i] = v
// 	}

// 	img := img{}

// 	if !vals[0].IsString() {
// 		return ctx.Error("img: src must be a string", nil)
// 	}

// 	img.src = vals[0].String()

// 	haveWorH := false
// 	if vals[1] != nil {
// 		if !vals[1].IsInteger() {
// 			return ctx.Error("img: width must be an integer", nil)
// 		}

// 		w := vals[1].Integer()
// 		if w < 0 {
// 			return ctx.Error("img: width must be greather than 0", nil)
// 		}

// 		haveWorH = true
// 		img.w = uint(w)
// 	}

// 	if vals[2] != nil {
// 		if !vals[2].IsInteger() {
// 			return ctx.Error("img: height must be an integer", nil)
// 		}

// 		h := vals[2].Integer()
// 		if h < 0 {
// 			return ctx.Error("img: height must be greather than 0", nil)
// 		}

// 		haveWorH = true
// 		img.h = uint(h)
// 	}

// 	if vals[3] != nil {
// 		if !vals[3].IsString() {
// 			return ctx.Error("img: crop must be a string", nil)
// 		}

// 		switch vals[2].String() {
// 		case "centered":
// 			img.crop = cropCentered

// 		case "left":
// 			img.crop = cropLeft

// 		default:
// 			return ctx.Error(
// 				fmt.Sprintf("img: unrecognized crop argument: %s", vals[2].String()),
// 				nil)
// 		}
// 	}

// 	if haveWorH && img.h == 0 && img.w == 0 {
// 		return ctx.Error("img: image width or height must be greater than 0", nil)
// 	}

// 	rc := s.getRelContent(c, fChangeExt(img.src, ""))
// 	if rc == nil {
// 		s.addError(c, fmt.Errorf("image not found: %s", img.src))
// 	} else {
// 		var err error

// 		ci, ok := rc.(*contentImg)
// 		if !ok {
// 			err = fmt.Errorf("img: %s is not an image and can't be scaled",
// 				img.src)
// 		}

// 		if err == nil {
// 			err = ci.scale(img)
// 		}

// 		if err != nil {
// 			s.addError(c, err)
// 		}
// 	}

// 	imgctx := p2.Context{
// 		"src":  img.src,
// 		"w":    img.w,
// 		"h":    img.h,
// 		"crop": img.crop,
// 	}

// 	lo := s.l.find(c.path(), "img_tag")
// 	err := lo.tpl.ExecuteWriter(imgctx, w)
// 	if err != nil {
// 		return ctx.Error(fmt.Sprintf("img: failed to write tag: %s", err), nil)
// 	}

// 	return nil
// }

// func imgTag(d *p2.Parser, s *p2.Token, args *p2.Parser) (p2.INodeTag, *p2.Error) {
// 	if args.Count() == 0 {
// 		return nil, args.Error("img: an image path is required", nil)
// 	}

// 	n := imgTagNode{}

// 	src, err := args.ParseExpression()
// 	if err != nil {
// 		return nil, err
// 	}

// 	n.src = src

// 	for args.Remaining() > 0 {
// 		t := args.MatchType(p2.TokenIdentifier)
// 		if t == nil {
// 			return nil, args.Error(
// 				fmt.Sprintf("img: unexpected token: expected %s, got %d",
// 					p2.TokenIdentifier,
// 					args.Current().Typ),
// 				t)
// 		}

// 		var where *p2.IEvaluator

// 		switch t.Val {
// 		case "crop":
// 			where = &n.crop

// 		case "height":
// 			where = &n.h

// 		case "width":
// 			where = &n.w
// 		}

// 		if where == nil {
// 			return nil, args.Error(
// 				fmt.Sprintf("img: unrecognized option: %s", t.Val),
// 				t)
// 		}

// 		arg := args.Match(p2.TokenSymbol, "=")
// 		if arg == nil {
// 			return nil, args.Error(
// 				fmt.Sprintf("img: %s requires an argument", t.Val),
// 				t)
// 		}

// 		opt, err := args.ParseExpression()
// 		if err != nil {
// 			return nil, err
// 		}

// 		*where = opt
// 	}

// 	return n, nil
// }
