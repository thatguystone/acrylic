package assets

import (
	"testing"

	"github.com/thatguystone/acrylic/internal/imgs"
	"github.com/thatguystone/acrylic/internal/pool"
	"github.com/thatguystone/acrylic/internal/test"
	"github.com/thatguystone/cog/check"
)

func TestMain(m *testing.M) {
	check.Main(m)
}

func newTest(t *testing.T) (*test.C, *A) {
	c := test.New(t)
	imgs := imgs.New(c.St)

	return c, New(imgs, c.St)
}

func testRender(t *testing.T, debug bool) {
	c, a := newTest(t)

	pool.Pool(&c.St.Run, func() {
		a.Render()
	})

	c.True(c.St.Errs.Ok())
}

func TestDebug(t *testing.T) {
	testRender(t, true)
}

func TestProd(t *testing.T) {
	testRender(t, false)
}
