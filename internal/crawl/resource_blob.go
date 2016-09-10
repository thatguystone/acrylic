package crawl

import (
	"io"
	"os"
	"time"
)

type resourceBlob struct {
	resourceBase
}

func (rsrc resourceBlob) recheck(r io.Reader) {}

func (rsrc resourceBlob) process(resp *response) io.Reader {
	// The client might use the build directory as a cache, so be sure to
	// check that the file actually needs to be updated before going through
	// the trouble of writing it out.

	info, err := os.Stat(rsrc.state.outputPath(rsrc.path()))

	alreadyUpdated := err == nil &&
		info.ModTime().Truncate(time.Second).Equal(resp.lastMod)
	if alreadyUpdated {
		return nil
	}

	return resp.Body
}
