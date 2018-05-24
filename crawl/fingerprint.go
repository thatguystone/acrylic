package crawl

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"path"

	"github.com/thatguystone/cog/cfs"
)

func fingerprint(r io.Reader) (fp string, err error) {
	sum, err := hashReader(r)
	if err == nil {
		fp = hex.EncodeToString(sum)
	}

	return
}

func addFingerprint(p, fp string) string {
	return cfs.ChangeExt(p, fmt.Sprintf("%s%s", fp, path.Ext(p)))
}

func hashReader(r io.Reader) (sum []byte, err error) {
	hash := sha1.New()
	_, err = io.Copy(hash, r)
	if err == nil {
		sum = hash.Sum(nil)
	}

	return
}
