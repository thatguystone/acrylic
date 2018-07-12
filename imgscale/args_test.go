package imgscale

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestArgsStrings(t *testing.T) {
	c := check.New(t)

	newInt := func(i int) *int { return &i }

	tests := []struct {
		args       args
		query      string
		nameSuffix string
	}{
		{},
		{
			args: args{
				W:   newInt(100),
				Ext: ".jpg",
			},
			nameSuffix: "-100x.jpg",
		},
		{
			args: args{
				H:   newInt(100),
				Ext: ".jpg",
			},
			nameSuffix: "-x100.jpg",
		},
		{
			args: args{
				W:   newInt(100),
				H:   newInt(100),
				Ext: ".jpg",
			},
			nameSuffix: "-100x100.jpg",
		},
		{
			args: args{
				Q:   newInt(50),
				Ext: ".jpg",
			},
			nameSuffix: "-q50.jpg",
		},
		{
			args: args{
				Crop: true,
				Ext:  ".jpg",
			},
			nameSuffix: "-c.jpg",
		},
		{
			args: args{
				Crop:    true,
				Gravity: northWest,
				Ext:     ".jpg",
			},
			nameSuffix: "-cnw.jpg",
		},
		{
			args: args{
				Ext: ".png",
			},
			nameSuffix: ".png",
		},
	}

	for _, test := range tests {
		c.Equal(test.args.nameSuffix(), test.nameSuffix)
	}
}
