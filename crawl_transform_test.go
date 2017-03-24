package acrylic

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/thatguystone/cog/check"
)

func testGetRel(u string) string {
	return "rel_" + u
}

func TestCrawlTransformBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `
			<link rel="stylesheet" href="/style.css">
			<img src="img.gif" srcset="img.gif 1x, img.gif 2x">
			<a href="#hash">hash</a>
			<a href="test#hash">test</a>
		`)
	})
	r.GET("/style.css", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/css")
		io.WriteString(w, `
			.body {
				background: url("/img.gif");
				background: -webkit-image-set(
					url("/img.gif") 1x,
					url("/img.gif") 2x);
				background: image-set(
					url("/img.gif") 1x,
					url("/img.gif") 2x);
			}
		`)
	})
	r.GET("/img.gif", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Write(gifBin)
	})
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `test`)
	})

	cr := Crawl{
		Handler: r,
		Output:  fs.Path("/"),
		Logf:    c.Logf,
	}
	c.Nil(cr.Do())
	c.Contains(fs.SReadFile("/index.html"), `href="#hash">hash`)
	c.Contains(fs.SReadFile("/index.html"), `href="test#hash">test`)
	c.Contains(fs.SReadFile("/style.css"), "url(/img.gif)")
}

func TestCrawlTransformHTMLRefs(t *testing.T) {
	c := check.New(t)

	html := strings.NewReader(`
		<a href="a.jpg"></a>
		<img src="img.jpg" srcset="img0.jpg 100w, img1.jpg 100w">`)
	buff := new(bytes.Buffer)
	err := transformHTMLRefs(html, buff, testGetRel)
	c.Must.Nil(err)
	c.Contains(buff.String(), "rel_a.jpg")
	c.Contains(buff.String(), "rel_img.jpg")
	c.Contains(buff.String(), "rel_img0.jpg")
	c.Contains(buff.String(), "rel_img1.jpg")
}

func TestCrawlTransformSrcSet(t *testing.T) {
	c := check.New(t)

	c.Equal(
		transformSrcSet(`s.jpg 1000w, a.jpg b c d e f, l.jpg 3000w`, testGetRel),
		`rel_s.jpg 1000w, rel_a.jpg b c d e f, rel_l.jpg 3000w`)
	c.Equal(
		transformSrcSet(`  s.jpg 1000w,  m.jpg  2000w , l.jpg 3000w`, testGetRel),
		`rel_s.jpg 1000w, rel_m.jpg 2000w , rel_l.jpg 3000w`)
	c.Equal(
		transformSrcSet(`s,,.jpg 1000w, m,.jpg 2000w`, testGetRel),
		`rel_s,,.jpg 1000w, rel_m,.jpg 2000w`)
}

func TestCrawlTransformCSSUrls(t *testing.T) {
	c := check.New(t)

	css := strings.NewReader(`
		body {
			background: url("body(.jpg");
		}

		a {
			background: url("a.jpg");
		}

		.test {
			background: url("test.jpg");
		}

		.image-set {
			background: image-set(
				url("set0.jpg") 1000w,
				url("set1.jpg") 2000w);
		}`)
	buff := new(bytes.Buffer)
	err := transformCSSUrls(css, buff, testGetRel)
	c.Must.Nil(err)
	c.Contains(buff.String(), `url("rel_body(.jpg")`)
	c.Contains(buff.String(), `url("rel_a.jpg")`)
	c.Contains(buff.String(), `url("rel_test.jpg")`)
	c.Contains(buff.String(), `url("rel_set0.jpg")`)
	c.Contains(buff.String(), `url("rel_set1.jpg")`)
}

func TestCrawlTransformJSON(t *testing.T) {
	c := check.New(t)

	r := strings.NewReader(`{	"test":	1	}`)
	buff := new(bytes.Buffer)
	err := transformJSON(nil, r, buff)
	c.Nil(err)

	c.Equal(buff.String(), `{"test":1}`)
}

func TestCrawlTransformErrors(t *testing.T) {
	c := check.New(t)

	c.NotNil(transformHTMLRefs(readSeeker{er: new(check.Errorer)}, nil, nil))
	c.NotNil(transformCSSUrls(readSeeker{er: new(check.Errorer)}, nil, nil))
}
