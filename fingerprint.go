package acrylic

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/thatguystone/cog/cfs"
)

func shortFingerprint(r io.Reader) (string, error) {
	fp, err := fingerprint(r)
	if err == nil {
		fp = fp[:14]
	}

	return fp, err
}

func fingerprint(r io.Reader) (fp string, err error) {
	sum, err := hashReader(r)
	if err == nil {
		fp = hex.EncodeToString(sum)
	}

	return
}

func addFingerprint(p, fp string) string {
	return cfs.ChangeExt(p, fmt.Sprintf(".%s%s", fp, path.Ext(p)))
}

func hashFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	return hashReader(f)
}

func hashReader(r io.Reader) (sum []byte, err error) {
	hash := sha1.New()
	_, err = io.Copy(hash, r)
	if err == nil {
		sum = hash.Sum(nil)
	}

	return
}
