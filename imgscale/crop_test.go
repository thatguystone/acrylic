package imgscale

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestCropGravities(t *testing.T) {
	c := check.New(t)

	gravities := []cropGravity{
		center,
		northWest,
		north,
		northEast,
		west,
		east,
		southWest,
		south,
		southEast,
	}

	for _, gravity := range gravities {
		c.NotPanics(func() {
			_ = gravity.String()
		})

		c.NotPanics(func() {
			_ = gravity.shortName()
		})

		b, err := gravity.MarshalText()
		c.Nil(err)

		var g cropGravity

		err = g.UnmarshalText(b)
		c.Nil(err)
		c.Equal(gravity, g)
	}
}

func TestCropGravityErrors(t *testing.T) {
	c := check.New(t)

	c.Panics(func() {
		_ = cropGravity(10000).String()
	})

	c.Panics(func() {
		_ = cropGravity(10000).shortName()
	})

	var g cropGravity
	err := g.UnmarshalText([]byte(`100000`))
	c.NotNil(err)
}
