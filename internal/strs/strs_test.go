package strs

import (
	"testing"
	"time"

	"github.com/thatguystone/cog/check"
)

func TestToDate(t *testing.T) {
	c := check.New(t)

	date, ok := ToDate("2015-07-21-post-title")
	c.MustTrue(ok)
	c.True(date.Equal(time.Date(2015, 7, 21, 0, 0, 0, 0, time.Local)))
}

func TestToTitle(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		in  string
		out string
	}{
		{
			in:  "post-title",
			out: "Post Title",
		},
		{
			in:  "post-with-title",
			out: "Post with Title",
		},
	}

	for _, test := range tests {
		c.Equal(ToTitle(test.in), test.out)
	}
}
