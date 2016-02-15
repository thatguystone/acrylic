package strs

import (
	"strings"
	"time"
	"unicode"
)

const (
	// DateFormat is the date format used everywhere
	DateFormat = "2006-01-02"
)

var (
	lowerTitleWords      = map[string]struct{}{}
	lowerTitleWordsSlice = []string{
		"a", "an", "the",
		"and", "but", "or", "for", "nor", "so", "with",
		"on", "at", "to", "from", "by",
		"etc", "etc.",
	}
)

func init() {
	for _, s := range lowerTitleWordsSlice {
		lowerTitleWords[s] = struct{}{}
	}
}

func titleSpaceChar(r rune) bool {
	return unicode.IsSpace(r) ||
		r == '-' ||
		r == '_'
}

// ToDate parses the date out of a post's filename
func ToDate(title string) (time.Time, bool) {
	if len(title) > len(DateFormat) {
		title = title[:len(DateFormat)]
	}

	t, err := time.ParseInLocation(DateFormat, title, time.Local)
	return t, err == nil
}

// ToTitle turns a post's filename into a post title
func ToTitle(title string) string {
	parts := strings.FieldsFunc(title, titleSpaceChar)

	for i, part := range parts {
		_, ok := lowerTitleWords[part]
		if i == 0 || !ok {
			parts[i] = strings.Title(part)
		}
	}

	return strings.Join(parts, " ")
}
