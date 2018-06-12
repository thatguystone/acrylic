package crawl

type linkRewrite map[string]string

func (lr linkRewrite) ResolveLink(link string) ResolvedLinker {
	to, ok := lr[link]
	if !ok {
		to = link
	}

	return rewrittenLink(to)
}

type rewrittenLink string

func (l rewrittenLink) Get() string {
	return string(l)
}
