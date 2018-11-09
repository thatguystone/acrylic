package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

func calcKey(s []string) string {
	h := sha256.New()

	for _, key := range s {
		// Quote each key to ensure that there are no collisions amongst
		// unrelated keys.
		key = strconv.Quote(key)
		h.Write([]byte(key))
	}

	return hex.EncodeToString(h.Sum(nil))
}
