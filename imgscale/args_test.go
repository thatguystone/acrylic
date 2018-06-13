package imgscale

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestArgsStrings(t *testing.T) {
	c := check.New(t)

	newInt := func(i int) *int { return &i }
	newStr := func(s string) *string { return &s }

	tests := []struct {
		args    args
		query   string
		variant string
	}{
		{
			variant: "img.jpg",
		},
		{
			args: args{
				W: newInt(100),
			},
			query:   "?W=100",
			variant: "img-100x.jpg",
		},
		{
			args: args{
				H: newInt(100),
			},
			query:   "?H=100",
			variant: "img-x100.jpg",
		},
		{
			args: args{
				W: newInt(100),
				H: newInt(100),
			},
			query:   "?H=100&W=100",
			variant: "img-100x100.jpg",
		},
		{
			args: args{
				Q: newInt(50),
			},
			query:   "?Q=50",
			variant: "img-q50.jpg",
		},
		{
			args: args{
				Crop: true,
			},
			query:   "?Crop=1",
			variant: "img-c.jpg",
		},
		{
			args: args{
				Crop:    true,
				Gravity: northWest,
			},
			query:   "?Crop=1&Gravity=nw",
			variant: "img-cnw.jpg",
		},
		{
			args: args{
				Ext: newStr(".png"),
			},
			query:   "?Ext=.png",
			variant: "img.png",
		},
	}

	for _, test := range tests {
		c.Equal(test.args.query(), test.query)
		c.Equal(test.args.variantName("img.jpg"), test.variant)
	}
}
