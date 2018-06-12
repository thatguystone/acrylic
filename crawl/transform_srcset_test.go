package crawl

import (
	"fmt"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestTransformSrcSetRewrites(t *testing.T) {
	c := check.New(t)

	const val = `/img.gif, /img.gif 2x, /img.gif 3x`

	lr := linkRewrite{
		"/img.gif": "img.hash.gif",
	}

	out := newSrcSetTransform(lr, val).get()
	c.Contains(out, "img.hash.gif")
	c.NotContains(out, "img.gif")
}

func TestParseSrcSet(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		in  string
		ss  srcSet
		str string
	}{
		{
			in:  "",
			ss:  nil,
			str: "",
		},
		{
			in:  " ",
			ss:  nil,
			str: "",
		},
		{
			in: "small.jpg",
			ss: srcSet{
				{
					url: "small.jpg",
				},
			},
			str: "small.jpg",
		},
		{
			in: "small.jpg 320w",
			ss: srcSet{
				{
					url: "small.jpg",
					descriptors: []string{
						"320w",
					},
				},
			},
			str: "small.jpg 320w",
		},
		{
			in: "small.jpg 320w, medium.jpg 480w",
			ss: srcSet{
				{
					url: "small.jpg",
					descriptors: []string{
						"320w",
					},
				},
				{
					url: "medium.jpg",
					descriptors: []string{
						"480w",
					},
				},
			},
			str: "small.jpg 320w, medium.jpg 480w",
		},
		{
			in: "small.jpg 320w,\n\t\tmedium.jpg 480w,\n\t\tlarge.jpg 800w",
			ss: srcSet{
				{
					url: "small.jpg",
					descriptors: []string{
						"320w",
					},
				},
				{
					url: "medium.jpg",
					descriptors: []string{
						"480w",
					},
				},
				{
					url: "large.jpg",
					descriptors: []string{
						"800w",
					},
				},
			},
			str: "small.jpg 320w, medium.jpg 480w, large.jpg 800w",
		},
		{
			in: "test, something 2x (parens),",
			ss: srcSet{
				{
					url: "test",
				},
				{
					url: "something",
					descriptors: []string{
						"2x",
						"(parens)",
					},
				},
			},
			str: "test, something 2x (parens)",
		},
		{
			in: "test,  something  2x(really  long  descriptor  ),     ",
			ss: srcSet{
				{
					url: "test",
				},
				{
					url: "something",
					descriptors: []string{
						"2x(really  long  descriptor  )",
					},
				},
			},
			str: "test, something 2x(really  long  descriptor  )",
		},
	}

	for i, test := range tests {
		test := test

		c.Run(fmt.Sprintf("%d-%s", i, test.in), func(c *check.C) {
			ss := parseSrcSet(test.in)
			c.Equal(ss, test.ss)
			c.Equal(ss.String(), test.str)
		})
	}
}
