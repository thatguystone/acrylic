package crawl

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
)

var reCSSURL = regexp.MustCompile(`url\("?(.*?)"?\)`)

type resourceCSS struct {
	resourceBase
}

func (rsrc *resourceCSS) process(resp *response) io.Reader {
	r := Minify.Reader("text/css", resp.Body)
	css, err := ioutil.ReadAll(r)
	resp.Body.Close()

	if err != nil {
		rsrc.state.Errorf("[css] failed to read from %s: %v",
			resp.Request.URL, err)
		return nil
	}

	matches := reCSSURL.FindAllSubmatch(css, -1)
	for _, match := range matches {
		url := string(match[1])
		c := rsrc.loadRelative(url)

		cURL := c.url.String()
		if url != cURL {
			css = bytes.Replace(css,
				match[0],
				[]byte(fmt.Sprintf(`url("%s")`, cURL)),
				-1)
		}
	}

	return bytes.NewReader(css)
}
