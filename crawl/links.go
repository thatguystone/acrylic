package crawl

type LinkConfig int

const (
	PreserveLinks LinkConfig = iota
	AbsoluteLinks
	RelativeLinks
)

// A LinkResolver resolves links asynchronously
type LinkResolver interface {
	ResolveLink(link string) ResolvedLinker

	// TODO(as): does anyone actually use <base href="">?
	// WithBase(base string) LinkResolver
}

// A ResolvedLinker gets the results of an async link resolve
type ResolvedLinker interface {
	Get() string
}

// linkResolver implements LinkResolver so that Page doesn't expose it (since
// it's technically only useful during a crawl, not after).
type linkResolver Page

func (lr *linkResolver) ResolveLink(link string) ResolvedLinker {
	pg := (*Page)(lr)

	relURL, err := pg.OrigURL.Parse(link)
	if err != nil {
		pg.addError(err)
		return &resolvedLink{
			invalid: link,
		}
	}

	return &resolvedLink{
		from: pg,
		to:   pg.cr.get(relURL),
		frag: relURL.Fragment,
	}
}

type resolvedLink struct {
	invalid string // Only set if couldn't parse link

	from *Page
	to   *Page
	frag string
}

func (rl *resolvedLink) Get() string {
	if rl.invalid != "" {
		return rl.invalid
	}

	to := rl.to.FollowRedirects()

	uu := to.URL
	uu.Fragment = rl.frag
	return uu.String()
}

// func getRelLinkTo() string {
// 	const up = "../"

// 	src := path.Clean(c.URL.Path)
// 	dst := path.Clean(o.URL.Path)

// 	start := 0
// 	for i := 0; i < len(src) && i < len(dst); i++ {
// 		if src[i] != dst[i] {
// 			break
// 		}

// 		if src[i] == '/' {
// 			start = i + 1
// 		}
// 	}

// 	var b strings.Builder
// 	dst = dst[start:]
// 	dirs := strings.Count(src[start:], "/")

// 	b.Grow((len(up) * dirs) + len(dst))
// 	for i := 0; i < dirs; i++ {
// 		b.WriteString(up)
// 	}

// 	b.WriteString(dst)

// 	return b.String()
// }