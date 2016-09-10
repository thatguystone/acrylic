package crawl

type contentType int

const (
	contentBlob contentType = iota
	contentExternal
	contentRedirect
	contentHTML
	contentCSS
	contentJS
)

func contentTypeFromMime(mime string) contentType {
	switch mime {
	case "text/html":
		return contentHTML

	case "text/css":
		return contentCSS

	case "application/javascript", "text/javascript":
		return contentJS

	default:
		return contentBlob
	}
}

func (ct contentType) newResource() resourcer {
	switch ct {
	case contentHTML:
		return new(resourceHTML)

	case contentCSS:
		return new(resourceCSS)

	case contentJS, contentBlob:
		return new(resourceBlob)

	default:
		return nil
	}
}

func (ct contentType) String() string {
	switch ct {
	case contentBlob:
		return "blob"

	case contentExternal:
		return "external"

	case contentRedirect:
		return "redirect"

	case contentHTML:
		return "html"

	case contentCSS:
		return "css"

	case contentJS:
		return "js"

	default:
		return "<invalid type>"
	}
}
