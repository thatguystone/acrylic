package file

import (
	"testing"
	"time"

	"github.com/thatguystone/acrylic/internal/test"
	"github.com/thatguystone/cog/check"
)

func TestMain(m *testing.M) {
	check.Main(m)
}

func touch(c *test.C, path string) string {
	c.FS.WriteFile(path, nil)
	return c.FS.Path(path)
}

func TestFileInfo(t *testing.T) {
	c := test.New(t)

	f := New(
		touch(c, "content/blog/img.jpg"),
		c.FS.Path("content/"),
		false, c.St)
	c.Equal(f.Dst, c.FS.Path("public/blog/img.jpg"))
	c.Equal(f.Cat, "blog")
	c.Equal(f.Name, "img")
	c.Equal(f.SortName, "img.jpg")
	c.Equal(f.URL, "/blog/img.jpg")
	c.True(f.Date.IsZero())

	f = New(
		touch(c, "content/blog/2015-07-21-test-post/img.jpg"),
		c.FS.Path("content/"),
		false, c.St)
	c.Equal(f.Dst, c.FS.Path("public/blog/2015/07/21/test-post/img.jpg"))
	c.Equal(f.Cat, "blog")
	c.Equal(f.Name, "test-post")
	c.Equal(f.SortName, "2015-07-21-test-post/img.jpg")
	c.Equal(f.URL, "/blog/2015/07/21/test-post/img.jpg")
	c.True(f.Date.Equal(time.Date(2015, 7, 21, 0, 0, 0, 0, time.Local)))

	f = New(
		touch(c, "content/blog/sub-cat/some-page.html"),
		c.FS.Path("content/"),
		false, c.St)
	c.Equal(f.Dst, c.FS.Path("public/blog/sub-cat/some-page.html"))
	c.Equal(f.Cat, "blog/sub-cat")
	c.Equal(f.Name, "some-page")
	c.Equal(f.SortName, "some-page.html")
	c.Equal(f.URL, "/blog/sub-cat/some-page.html")
	c.True(f.Date.IsZero())
}

func TestPageFileInfo(t *testing.T) {
	c := test.New(t)

	f := New(
		touch(c, "content/blog/2015-07-21-test-post/index.html"),
		c.FS.Path("content/"),
		true, c.St)
	c.Equal(f.Dst, c.FS.Path("public/blog/2015/07/21/test-post/index.html"))
	c.Equal(f.Cat, "blog")
	c.Equal(f.Name, "test-post")
	c.Equal(f.SortName, "2015-07-21-test-post")
	c.Equal(f.URL, "/blog/2015/07/21/test-post/")
	c.True(f.Date.Equal(time.Date(2015, 7, 21, 0, 0, 0, 0, time.Local)))

	f = New(
		touch(c, "content/index.html"),
		c.FS.Path("content/"),
		true, c.St)
	c.Equal(f.Dst, c.FS.Path("public/index.html"))
	c.Equal(f.Cat, "")
	c.Equal(f.Name, "index")
	c.Equal(f.SortName, "index.html")
	c.Equal(f.URL, "/index.html")
	c.True(f.Date.IsZero())

	f = New(
		touch(c, "blog/page1.html"),
		c.FS.Path(""),
		true, c.St)
	c.Equal(f.Dst, c.FS.Path("public/blog/page1/index.html"))
	c.Equal(f.Cat, "blog")
	c.Equal(f.Name, "page1")
	c.Equal(f.SortName, "page1.html")
	c.Equal(f.URL, "/blog/page1/")
	c.True(f.Date.IsZero())

	f = New(
		touch(c, "content/blog/index.html"),
		c.FS.Path("content/"),
		true, c.St)
	c.Equal(f.Dst, c.FS.Path("public/blog/index.html"))
	c.Equal(f.Cat, "blog")
	c.Equal(f.Name, "index")
	c.Equal(f.SortName, "index.html")
	c.Equal(f.URL, "/blog/")
	c.True(f.Date.IsZero())
}
