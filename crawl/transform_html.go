package crawl

import (
	"bytes"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func transformHTML(lr LinkResolver, b []byte) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	var cbs []func()

	var visit func(parent, n *html.Node)
	visit = func(parent, n *html.Node) {
		for cn := n.FirstChild; cn != nil; cn = cn.NextSibling {
			visit(n, cn)
		}

		if parent != nil && parent.DataAtom == atom.Style {
			tf := newCSSTransform(lr, n.Data)
			cbs = append(cbs, func() {
				n.Data = string(tf.get())
			})
		}

		if n.Type != html.ElementNode {
			return
		}

		for i := range n.Attr {
			attr := &n.Attr[i]

			switch attr.Key {
			case "src", "href":
				res := lr.ResolveLink(attr.Val)
				cbs = append(cbs, func() {
					attr.Val = res.Get()
				})

			case "srcset":
				tf := newSrcSetTransform(lr, attr.Val)
				cbs = append(cbs, func() {
					attr.Val = tf.get()
				})

			case "style":
				tf := newCSSTransform(lr, attr.Val)
				cbs = append(cbs, func() {
					attr.Val = string(tf.get())
				})
			}
		}
	}

	visit(nil, doc)
	for _, cb := range cbs {
		cb()
	}

	var buff bytes.Buffer
	err = html.Render(&buff, doc)
	if err != nil {
		return nil, err
	}

	return Minify.Bytes(htmlType, buff.Bytes())
}
