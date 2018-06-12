package crawl

import (
	"mime"
	"path/filepath"
)

const (
	// DefaultType is the default content type that servers typically send back
	// when they can't determine a file's type
	DefaultType = "application/octet-stream"

	htmlType = "text/html"
	cssType  = "text/css"
	jsType   = "application/javascript"
	jsonType = "application/json"
	svgType  = "image/svg+xml"
)

// checkServeMime checks that the file type that a static server will respond
// with for the generated file is consistent with the type that was originally
// sent back.
func checkServeMime(path, respMediaType string) error {
	ext := filepath.Ext(path)

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = DefaultType
	}

	mediaType, _, _ := mime.ParseMediaType(mimeType)

	if respMediaType != mediaType {
		return MimeTypeMismatchError{
			Ext:          ext,
			Guess:        mediaType,
			FromResponse: respMediaType,
		}
	}

	return nil
}
