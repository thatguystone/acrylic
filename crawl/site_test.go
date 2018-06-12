package crawl

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestCleanPath(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		in, out string
	}{
		{"", ""},
		{"/", "/"},
		{"/test/", "/test/"},
		{"/test/../", "/"},
		{"/test/..", "/"},
	}

	for _, test := range tests {
		c.Equal(cleanPath(test.in), test.out)
	}
}
