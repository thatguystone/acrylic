package crawl

import (
	"strings"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestFingerprintBasic(t *testing.T) {
	c := check.New(t)

	fp, err := fingerprint(strings.NewReader("test"))
	c.Nil(err)
	c.Equal(fp, "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")
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
