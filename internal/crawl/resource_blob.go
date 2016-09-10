package crawl

import (
	"io"
	"os"
)

type resourceBlob struct {
	resourceBase
}

func (rsrc resourceBlob) recheck(resp *response, f *os.File) error {
	// Nothing to do: can't recheck a blob
	return nil
}

func (rsrc resourceBlob) process(resp *response, f *os.File) error {
	// The client might use the build directory as a cache, so be sure to
	// check that the file actually needs to be updated before going through
	// the trouble of writing it out.
	info, err := f.Stat()
	if err != nil {
		return err
	}

	mod := info.ModTime()
	if mod.Equal(resp.lastMod) {
		return nil
	}

	_, err = io.Copy(f, resp.Body)
	return err
}
