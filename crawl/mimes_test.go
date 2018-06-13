package crawl

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestCheckMime(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		path     string
		contType string
		err      error
	}{
		{
			path:     "/test.html",
			contType: htmlType,
			err:      nil,
		},
		{
			path:     "/test.css",
			contType: cssType,
			err:      nil,
		},
		{
			path:     "/test.js",
			contType: jsType,
			err:      nil,
		},
		{
			path:     "/test.json",
			contType: jsonType,
			err:      nil,
		},
		{
			path:     "/test.svg",
			contType: svgType,
			err:      nil,
		},
		{
			path:     "/test",
			contType: DefaultType,
			err:      nil,
		},
		{
			path:     "/test.not-a-type",
			contType: DefaultType,
			err:      nil,
		},
		{
			path:     "/test.not-a-type",
			contType: htmlType,
			err: MimeTypeMismatchError{
				Ext:          ".not-a-type",
				Guess:        DefaultType,
				FromResponse: htmlType,
			},
		},
	}

	for _, test := range tests {
		err := checkServeMime(test.path, test.contType)
		c.Equal(err, test.err)

		if err != nil && test.err != nil {
			c.Equal(err.Error(), test.err.Error())
		}
	}
}
