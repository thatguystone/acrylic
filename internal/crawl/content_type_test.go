package crawl

import (
	"strings"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestContentTypeString(t *testing.T) {
	check.New(t)

	i := 0
	str := ""
	for !strings.Contains(str, "invalid") {
		str = contentType(i).String()
		i++
	}
}
