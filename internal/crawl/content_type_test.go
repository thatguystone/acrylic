package crawl

import (
	"strings"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestContentTypeCoverage(t *testing.T) {
	check.New(t)

	i := 0
	str := ""
	for !strings.Contains(str, "invalid") {
		ct := contentType(i)
		ct.newResource()
		str = ct.String()
		i++
	}
}
