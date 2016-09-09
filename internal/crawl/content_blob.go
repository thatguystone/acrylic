package crawl

import (
	"net/http"
	"os"
)

func (c *content) processBlob(resp *http.Response) {
	// The client might use the build directory as a cache, so be sure to
	// check that the file actually needs to be updated before going through
	// the trouble of writing it out.
	path, _ := c.outputPath()
	info, err := os.Stat(path)
	if err == nil && info.ModTime().Equal(c.lastMod) {
		return
	}

	c.saveReader(resp.Body)
	resp.Body.Close()
}
