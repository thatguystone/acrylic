package crawl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
)

var reCSSURL = regexp.MustCompile(`url\("?(.*?)"?\)`)

func (c *content) processCSS(resp *http.Response) {
	proc := processCSS{
		process: process{
			content: c,
		},
	}

	proc.run(resp)
}

type processCSS struct {
	process
}

func (proc *processCSS) run(resp *http.Response) {
	r := Minify.Reader("text/css", resp.Body)
	css, err := ioutil.ReadAll(r)
	resp.Body.Close()

	if err != nil {
		proc.state.Errorf("[html] failed to read css from %s: %v", proc, err)
		return
	}

	matches := reCSSURL.FindAllSubmatch(css, -1)
	for _, match := range matches {
		url := string(match[1])
		c := proc.loadRelative(url)

		cURL := c.url.String()
		if url != cURL {
			css = bytes.Replace(css,
				match[0],
				[]byte(fmt.Sprintf(`url("%s")`, cURL)),
				-1)
		}
	}

	proc.saveBytes(css)
}
