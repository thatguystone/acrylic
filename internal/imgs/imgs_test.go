package imgs

import (
	"fmt"
	"sort"
	"testing"

	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/pool"
	"github.com/thatguystone/acrylic/internal/test"
	"github.com/thatguystone/cog/check"
)

func TestMain(m *testing.M) {
	check.Main(m)
}

func newTest(t *testing.T) (*test.C, *Imgs) {
	c := test.New(t)
	is := New(c.St)

	return c, is
}

func TestBasic(t *testing.T) {
	c, is := newTest(t)

	c.FS.WriteFile("pic.gif", test.GifBin)

	f := file.New(c.FS.Path("pic.gif"), c.FS.Path(""), false, c.St)
	err := is.Load(f, true)
	c.MustNotError(err)

	// Meta file created after load?
	c.FS.FileExists("pic.gif.meta")
}

func TestMetadata(t *testing.T) {
	c, is := newTest(t)

	c.FS.SWriteFile("pic.gif.meta", "---\ntitle: bleep bloop\n---\n")
	c.FS.WriteFile("pic.gif", test.GifBin)

	path := c.FS.Path("pic.gif")
	f := file.New(path, c.FS.Path(""), false, c.St)
	err := is.Load(f, true)
	c.MustNotError(err)

	img := is.Get(path)
	c.MustNotEqual(img, nil)
	c.Equal(img.Title, "bleep bloop")
	c.Equal(img.inGallery, true)
}

func TestNoGalleryMeta(t *testing.T) {
	c, is := newTest(t)

	c.FS.SWriteFile("pic.gif.meta", "---\ngallery: false\n---\n")
	c.FS.WriteFile("pic.gif", test.GifBin)

	path := c.FS.Path("pic.gif")
	f := file.New(path, c.FS.Path(""), false, c.St)
	err := is.Load(f, true)
	c.MustNotError(err)

	img := is.Get(path)
	c.MustNotEqual(img, nil)
	c.Equal(img.inGallery, false)
}

func TestNoGalleryNotContent(t *testing.T) {
	c, is := newTest(t)

	c.FS.WriteFile("pic.gif", test.GifBin)

	path := c.FS.Path("pic.gif")
	f := file.New(path, c.FS.Path(""), false, c.St)
	err := is.Load(f, false)
	c.MustNotError(err)

	img := is.Get(path)
	c.MustNotEqual(img, nil)
	c.Equal(img.inGallery, false)
}

func TestMetadataErrors(t *testing.T) {
	c, is := newTest(t)

	c.FS.SWriteFile("pic.gif.meta", "--title bleep bloop\n---\n")
	c.FS.WriteFile("pic.gif", test.GifBin)

	f := file.New(c.FS.Path("pic.gif"), c.FS.Path(""), false, c.St)
	err := is.Load(f, true)
	c.Error(err)
}

func TestAllLoaded(t *testing.T) {
	c, is := newTest(t)

	for i := 10; i > 0; i-- {
		name := fmt.Sprintf("%d.gif", i)
		c.FS.WriteFile(name, test.GifBin)

		f := file.New(c.FS.Path(name), c.FS.Path(""), false, c.St)
		err := is.Load(f, i%2 == 0)
		c.MustNotError(err)
	}

	c.MustFalse(sort.IsSorted(is.all))
	for _, ims := range is.byCat {
		c.MustFalse(sort.IsSorted(ims))
	}

	pool.Pool(&c.St.Run, func() {
		is.AllLoaded()
	})

	c.MustTrue(sort.IsSorted(is.all))
	for _, ims := range is.byCat {
		c.MustTrue(sort.IsSorted(ims))
	}

	ims := is.All(true)
	c.Logf("%+v", ims)
	c.Len(ims, 5)
	c.MustTrue(sort.IsSorted(sort.Reverse(sort.StringSlice(ims))))
}
