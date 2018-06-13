package crawl

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestCleanURLPath(t *testing.T) {
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
		c.Equal(cleanURLPath(test.in), test.out)
	}
}
