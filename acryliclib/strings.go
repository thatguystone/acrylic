package acryliclib

import (
	"strings"
	"time"
	"unicode"
)

const (
	sDateFormat = "2006-01-02"
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

func sToDate(title string) (time.Time, bool) {
	if len(title) > len(sDateFormat) {
		title = title[:len(sDateFormat)]
	}

	t, err := time.Parse(sDateFormat, title)
	return t, err == nil
}

func sToTitle(title string) string {
	parts := strings.FieldsFunc(title, titleSpaceChar)

	for i, part := range parts {
		_, ok := lowerTitleWords[part]
		if i == 0 || !ok {
			parts[i] = strings.Title(part)
		}
	}

	return strings.Join(parts, " ")
}

func ssLast(ss []string) string {
	l := len(ss)
	if l == 0 {
		return ""
	}

	return ss[l-1]
}
