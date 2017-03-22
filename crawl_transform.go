package acrylic

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type transformer func(c *Content, r io.Reader, w *bytes.Buffer) error

var transforms map[string]transformer

func init() {
	// When initialized directly, go complains about an initialization loop
	transforms = map[string]transformer{
		"text/html": transformHTML,
		"text/css":  transformCSS,
		// "text/javascript":        transformJS,
		// "application/javascript": transformJS,
		"application/json": transformJSON,
	}
}

func transformHTML(c *Content, r io.Reader, w *bytes.Buffer) error {
	return transformHTMLRefs(r, w, func(u string) string {
		return c.getPathTo(u)
	})
}

func transformHTMLRefs(r io.Reader, w *bytes.Buffer, getRel func(string) string) error {
	r = Minify.Reader("text/html", r)
	doc, err := html.Parse(r)
	if err != nil {
		return err
	}

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}

		if n.Type != html.ElementNode {
			return
		}

		for i, attr := range n.Attr {
			switch attr.Key {
			case "src", "href":
				n.Attr[i].Val = getRel(attr.Val)

			case "srcset":
				n.Attr[i].Val = transformSrcSet(attr.Val, getRel)
			}
		}
	}
	visit(doc)

	return html.Render(w, doc)
}

func transformSrcSet(val string, getRel func(string) string) string {
	fields := strings.Fields(val)

	wantURL := true
	for i, field := range fields {
		if wantURL {
			fields[i] = getRel(field)
			wantURL = false
		} else if strings.HasSuffix(field, ",") {
			wantURL = true
		}
	}

	return strings.Join(fields, " ")
}

func transformCSS(c *Content, r io.Reader, w *bytes.Buffer) error {
	return transformCSSUrls(r, w, func(u string) string {
		return c.getPathTo(u)
	})
}

var reCSSURL = regexp.MustCompile(`url\("?(.*?)"?\)`)

func transformCSSUrls(r io.Reader, w *bytes.Buffer, getRel func(string) string) error {
	r = Minify.Reader("text/css", r)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	css := string(b)
	matches := reCSSURL.FindAllStringSubmatch(css, -1)
	replaces := make([]string, 0, len(matches)*2)

	for _, match := range matches {
		url := match[1]
		rel := getRel(url)

		if url != rel {
			replaces = append(replaces, match[0], fmt.Sprintf(`url("%s")`, rel))
		}
	}

	strings.NewReplacer(replaces...).WriteString(w, css)
	return nil
}

func transformJSON(c *Content, r io.Reader, w *bytes.Buffer) error {
	return Minify.Minify("application/json", w, r)
}
