package toner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	p2 "github.com/flosch/pongo2"
	"gopkg.in/yaml.v2"
)

type content struct {
	s   *site
	f   file
	err error

	rend       renderer
	rawContent []byte
	content    []byte
	tpla       tplAssets
	meta       data
	tags       []*tag
}

var (
	metaDelim         = []byte("---")
	bannedContentTags = []string{
		"js_tags",
		"css_tags",
	}
)

func (c *content) preprocess() error {
	c.tpla.assets = &c.s.a
	c.meta = data{}
	c.rend = getRenderer(c)

	c.f.dstPath = filepath.Join(c.s.cfg.PublicDir, fDropFirst(c.f.srcPath))
	c.f.dstPath = fChangeExt(c.f.dstPath, c.rend.ext(c))

	if !c.rend.renderable() {
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

	c.rawContent, err = c.executeTemplate(c.rawContent)
	return err
}

func (c *content) extractMeta() error {
	if !bytes.HasPrefix(c.rawContent, metaDelim) {
		return nil
	}

	end := bytes.Index(c.rawContent[3:], metaDelim)
	if end == -1 {
		return errors.New("metadata missing closing `---`")
	}

	meta := bytes.TrimSpace(c.rawContent[3 : end+3])
	c.rawContent = c.rawContent[end+7:]

	err := yaml.Unmarshal(meta, &c.meta)
	if err != nil {
		return err
	}

	return nil
}

func (c *content) sourceChanged() bool {
	dstat, err := c.s.fs.Stat(c.f.dstPath)
	if err != nil {
		return true
	}

	return !dstat.ModTime().Equal(c.f.info.ModTime())
}

func (c *content) executeTemplate(rc []byte) ([]byte, error) {
	// Run in total isolation from everything: content shouldn't be able to
	// modify layouts.
	set := p2.NewSet("temp")

	for _, t := range bannedContentTags {
		set.BanTag(t)
	}

	tpl, err := set.FromString(string(rc))
	if err != nil {
		return nil, err
	}

	return tpl.ExecuteBytes(c.getContext())
}

func (c *content) render() error {
	if !c.rend.renderable() {
		if !c.sourceChanged() {
			return nil
		}

		fr, err := c.s.fs.Open(c.f.srcPath)
		if err != nil {
			return err
		}

		defer fr.Close()

		fw, err := c.s.fCreate(c.f.dstPath)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %s", err)
		}

		defer fw.Close()

		_, err = io.Copy(fw, fr)
		return err
	}

	rc, err := c.rend.render(c)
	if err != nil {
		return err
	}

	p, _ := filepath.Split(fDropFirst(c.f.srcPath))
	lo := c.s.l.find(p, "_single")

	fw, err := c.s.fCreate(c.f.dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %s", err)
	}

	defer fw.Close()

	c.tpla.append(&lo.tpla)

	return lo.execute(c.s, c.getContext(), rc, fw)
}

func (c *content) getContext() p2.Context {
	return p2.Context{
		assetsKey:  &c.tpla,
		relPathKey: filepath.Dir(c.f.srcPath),
	}
}
