package pages

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/flosch/pongo2"
	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/min"
	"github.com/thatguystone/acrylic/internal/state"
	"github.com/thatguystone/acrylic/internal/strs"
	"github.com/thatguystone/cog/cfs"
)

const (
	scissors     = `<!-- >8 acrylic-content -->`
	scissorsEnd  = `<!-- acrylic-content 8< -->`
	moreScissors = `<!--more-->`
)

type P struct {
	file.F
	st         *state.S
	Content    string
	Summary    string
	isListPage bool
}

// Used to break the circular import between tmpl and pages
type TmplCompiler interface {
	Compile(tmpl string) (TmplRenderer, error)
}

type TmplRenderer interface {
	Render(f file.F, extraVars pongo2.Context) (string, error)
}

var (
	frontMatterStart = []byte("---\n")
	frontMatterEnd   = []byte("\n---\n")
)

func newP(st *state.S, f file.F) (*P, error) {
	c, err := ioutil.ReadFile(f.Src)
	if err != nil {
		return nil, err
	}

	fm := map[string]interface{}{}
	if bytes.HasPrefix(c, frontMatterStart) {
		end := bytes.Index(c, frontMatterEnd)
		if end == -1 {
			return nil, fmt.Errorf("missing front matter end")
		}

		fmb := c[len(frontMatterStart):end]
		err = yaml.Unmarshal(fmb, &fm)
		if err != nil {
			return nil, fmt.Errorf("invalid front matter: %v", err)
		}

		c = c[end+len(frontMatterEnd):]
	}

	title := ""
	if t, ok := fm["title"].(string); ok {
		title = t
	} else {
		title = strs.ToTitle(f.Name)
	}

	isListPage := false
	if b, ok := fm["list_page"].(bool); ok {
		isListPage = b
	}

	pg := &P{
		F:          f,
		st:         st,
		isListPage: isListPage,
	}

	pg.Title = title
	pg.Content = string(c)
	pg.Meta = fm

	return pg, nil
}

func (p *P) Render(tc TmplCompiler) error {
	var content string

	rr, err := tc.Compile(p.Content)
	if err == nil {
		content, err = rr.Render(p.F, pongo2.Context{
			"page": p,
		})
	}

	if err != nil {
		return err
	}

	start := strings.Index(content, scissors)
	end := strings.Index(content, scissorsEnd)

	p.Content = ""
	if start >= 0 && end >= 0 {
		p.Content = content[start+len(scissors) : end]

		end := strings.Index(p.Content, moreScissors)
		if end >= 0 {
			p.Summary = p.Content[:end]
		}
	}

	return p.writeTo(p.Dst, content)
}

func (p *P) RenderList(tc TmplCompiler, pages []*P) error {
	rr, err := tc.Compile(p.Content)
	if err != nil {
		return err
	}

	total := len(pages)
	pageCount := int(math.Ceil(float64(total) / float64(p.st.Cfg.PerPage)))

	for i := 0; i < pageCount; i++ {
		listStart := i * p.st.Cfg.PerPage
		listEnd := listStart + p.st.Cfg.PerPage

		if listEnd > total {
			listEnd = total
		}

		content, err := rr.Render(p.F, pongo2.Context{
			"page":        p,
			"pages":       pages[listStart:listEnd],
			"pageNum":     i + 1,
			"listHasNext": i < (pageCount - 1),
		})
		if err != nil {
			return err
		}

		dst := p.Dst
		if i > 0 {
			dst = filepath.Join(
				p.st.Cfg.PublicDir,
				filepath.Dir(p.URL),
				"page", fmt.Sprintf("%d", i+1),
				"index.html")
		}

		err = p.writeTo(dst, content)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *P) writeTo(dst, content string) error {
	p.st.Unused.Used(dst)

	if !p.st.Cfg.Debug {
		b := bytes.Buffer{}
		err := min.Minify("text/html", &b, strings.NewReader(content))
		if err != nil {
			return err
		}

		content = b.String()
	}

	return cfs.Write(dst, []byte(content))
}
