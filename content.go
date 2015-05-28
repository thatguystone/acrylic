package toner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type content struct {
	s   *site
	f   file
	err error

	rend       renderer
	rawContent []byte
	content    []byte
	meta       data
	k          *kind
	tags       []*tag
}

var (
	metaDelim         = []byte("---")
	errMissingMetaEnd = errors.New("metadata missing closing `---`")
)

func (c *content) preprocess() error {
	c.meta = data{}
	c.rend = getRenderer(c)

	if !c.rend.templatable() {
		return nil
	}

	f, err := c.s.fs.Open(c.f.srcPath)
	if err != nil {
		return err
	}

	defer f.Close()

	c.rawContent, err = ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	err = c.extractMeta()
	if err != nil {
		return err
	}

	c.setType()

	// Strip away first directory
	c.f.dstPath = filepath.Join(
		c.s.cfg.PublicDir,
		c.f.srcPath[strings.IndexRune(c.f.srcPath[1:], os.PathSeparator)+1:])

	// Replace extension
	ext := filepath.Ext(c.f.dstPath)
	c.f.dstPath = c.f.dstPath[0:len(c.f.dstPath)-len(ext)] + c.rend.ext(c)

	c.rawContent, err = c.runTemplate(c.rawContent)
	return err
}

func (c *content) extractMeta() error {
	if !bytes.HasPrefix(c.rawContent, metaDelim) {
		return nil
	}

	end := bytes.Index(c.rawContent[3:], metaDelim)
	if end == -1 {
		return errMissingMetaEnd
	}

	meta := bytes.TrimSpace(c.rawContent[3 : end+3])
	c.rawContent = c.rawContent[end+7:]

	err := yaml.Unmarshal(meta, &c.meta)
	if err != nil {
		return err
	}

	return nil
}

func (c *content) setType() {

}

func (c *content) matchesDst() bool {
	dstat, err := c.s.fs.Stat(c.f.dstPath)
	if err != nil {
		return false
	}

	return dstat.ModTime().Equal(c.f.info.ModTime())
}

func (c *content) runTemplate(rc []byte) ([]byte, error) {
	return rc, nil
}

func (c *content) render() error {
	if !c.rend.templatable() {
		if c.matchesDst() {
			return nil
		}

		fr, err := c.s.fs.Open(c.f.srcPath)
		if err != nil {
			return err
		}

		defer fr.Close()

		fw, err := c.s.fs.OpenFile(c.f.dstPath, createFlags, c.s.cfg.FileMode)
		if err != nil {
			return err
		}

		defer fw.Close()

		_, err = io.Copy(fw, fr)
		return err
	}

	rc, err := c.rend.render(c)
	if err != nil {
		return err
	}

	c.content, err = c.runTemplate(rc)
	if err != nil {
		return err
	}

	fw, err := c.s.fcreate(c.f.dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %s", err)
	}

	defer fw.Close()
	_, err = fw.Write(rc)

	return err
}
