package crawl

import "net/url"

type process struct {
	*content
	baseURL *url.URL // If not set, uses content.url
}

func (proc process) loadRelative(sURL string) *content {
	base := proc.baseURL
	if base == nil {
		base = proc.url
	}

	url, err := base.Parse(sURL)
	if err != nil {
		proc.state.Errorf("[rel url] invalid URL %s: %v", sURL, err)
		return nil
	}

	c := proc.state.load(url.String())
	return c.follow()
}
