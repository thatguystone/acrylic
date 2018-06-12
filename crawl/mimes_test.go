package crawl

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

const gifType = "image/gif"

var gifBin = []byte{
	0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80,
	0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01,
	0x00, 0x3b,
}

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
