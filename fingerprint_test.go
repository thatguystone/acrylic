package acrylic

import (
	"strings"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestFingerprintBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fp, err := shortFingerprint(strings.NewReader("test"))
	c.Nil(err)
	c.Equal(fp, "a94a8fe5ccb19b")

	fs.SWriteFile("test", "test")
	_, err = hashFile(fs.Path("test"))
	c.Nil(err)
}

func TestFingerprintAdd(t *testing.T) {
	c := check.New(t)

	c.Equal(
		"test.abcd1234.ext",
		addFingerprint("test.ext", "abcd1234"))

	c.Equal(
		"test.tar.abcd1234.gz", // Unfortunate :(
		addFingerprint("test.tar.gz", "abcd1234"))
}

func TestFingerprintErrors(t *testing.T) {
	c := check.New(t)

	_, err := hashFile("doesnotexist")
	c.NotNil(err)
}
