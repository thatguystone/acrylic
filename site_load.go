package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/thatguystone/acrylic/internal/afs"
	"github.com/thatguystone/acrylic/internal/strs"
	"github.com/thatguystone/cog/cfs"
	"gopkg.in/yaml.v2"
)

var (
	frontMatterStart = []byte("---\n")
	frontMatterEnd   = []byte("\n---\n")
)

func (ss *siteState) walk(dir string, cb func(string, os.FileInfo)) {
	dir = filepath.Join(ss.baseDir, dir)

	if exists, _ := cfs.DirExists(dir); !exists {
		return
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ss.errs.add(path, err)
			return nil
		}

		ss.pool.Do(func() {
			cb(path, info)
		})

		return nil
	})
}

func (ss *siteState) loadData(file string, info os.FileInfo) {
	if info.IsDir() {
		return
	}

	var data []byte
	var err error

	name := afs.DropRoot(ss.baseDir, ss.cfg.DataDir, file)
	cached := filepath.Join(ss.baseDir, ss.cfg.CacheDir, "data", name)

	exists, _ := cfs.FileExists(cached)
	if exists && !afs.SrcChanged(file, cached) {
		data, err = ioutil.ReadFile(cached)
		if err != nil {
			ss.errs.add(file, err)
			return
		}

		until := time.Unix(int64(binary.BigEndian.Uint64(data[:8])), 0)
		if until.Before(time.Now()) {
			data = nil
		} else {
			data = data[8:]
		}
	}

	if len(data) == 0 {
		if (info.Mode() & 0111) != 0 {
			cmd := exec.Command(file)

			ob := bytes.Buffer{}
			cmd.Stdout = &ob

			eb := bytes.Buffer{}
			cmd.Stderr = &eb

			err := cmd.Run()
			if err != nil || eb.Len() > 0 {
				ss.errs.add(file, fmt.Errorf("execute failed: %v: %s",
					err,
					eb.String()))
				return
			}

			data = ob.Bytes()
		} else {
			data, err = ioutil.ReadFile(file)
			if err != nil {
				ss.errs.add(file, err)
				return
			}
		}
	}

	var v interface{}
	err = json.Unmarshal(data, &v)
	if err != nil {
		ss.errs.add(file, err)
		return
	}

	if v, ok := v.(map[string]interface{}); ok {
		if until, ok := v["acrylic_expires"].(float64); ok {
			b := bytes.Buffer{}
			binary.Write(&b, binary.BigEndian, uint64(until))
			b.Write(data)

			err = cfs.Write(cached, b.Bytes())
			if err != nil {
				ss.errs.add(file, fmt.Errorf("failed to write cache file: %v", err))
				return
			}

			os.Chtimes(cached, info.ModTime(), info.ModTime())
		}

		delete(v, "acrylic_expires")
	}

	ss.mtx.Lock()
	ss.data[name] = v
	ss.mtx.Unlock()
}

func (ss *siteState) loadContent(file string, info os.FileInfo) {
	if info.IsDir() {
		return
	}

	switch filepath.Ext(file) {
	case ".html":
		ss.loadPage(file, info)

	case ".jpg", ".gif", ".png", ".svg":
		ss.loadImg(file, info, true)

	case ".meta":
		// Ignore these

	default:
		ss.loadBlob(file, info)
	}
}

func (ss *siteState) loadAssetImages(file string, info os.FileInfo) {
	if !info.IsDir() {
		switch filepath.Ext(file) {
		case ".jpg", ".gif", ".png", ".svg":
			ss.loadImg(file, info, false)
		}
	}
}

func (ss *siteState) loadPage(file string, info os.FileInfo) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		ss.errs.add(file, err)
		return
	}

	fm := map[string]interface{}{}
	if bytes.HasPrefix(c, frontMatterStart) {
		end := bytes.Index(c, frontMatterEnd)
		if end == -1 {
			ss.errs.add(file, fmt.Errorf("missing front matter end"))
			return
		}

		fmb := c[len(frontMatterStart):end]
		err = yaml.Unmarshal(fmb, &fm)
		if err != nil {
			ss.errs.add(file, err)
			return
		}

		c = c[end+len(frontMatterEnd):]
	}

	cat, date, name, sortName, url, dst := ss.getOutInfo(file, ss.cfg.ContentDir, true)

	title := ""
	if t, ok := fm["title"].(string); ok {
		title = t
	} else {
		title = strs.ToTitle(name)
	}

	isListPage := false
	if b, ok := fm["list_page"].(bool); ok {
		isListPage = b
	}

	ss.addPage(&page{
		src:        file,
		dst:        dst,
		sortName:   sortName,
		Cat:        cat,
		Title:      title,
		Date:       date,
		Content:    string(c),
		URL:        url,
		isListPage: isListPage,
		Meta:       fm,
	})
}

func (ss *siteState) loadImg(file string, info os.FileInfo, isContent bool) {
	rootDir := ""
	if isContent {
		rootDir = ss.cfg.ContentDir
	}

	category, date, _, sortName, url, dst := ss.getOutInfo(file, rootDir, false)

	metaFile := file + ".meta"
	fm := map[string]interface{}{}
	if exists, _ := cfs.FileExists(metaFile); exists {
		b, err := ioutil.ReadFile(metaFile)
		if err != nil {
			ss.errs.add(file, err)
			return
		}

		err = yaml.Unmarshal(b, &fm)
		if err != nil {
			ss.errs.add(file, err)
			return
		}
	} else if strings.HasPrefix(file, ss.cfg.ContentDir) {
		cfs.Write(metaFile, []byte("---\ntitle: \n---\n"))
	}

	title := ""
	if t, ok := fm["title"].(string); ok {
		title = t
	}

	inGallery := isContent
	if g, ok := fm["gallery"].(bool); ok {
		inGallery = g
	}

	ss.addImage(&image{
		ss:        ss,
		src:       file,
		dst:       dst,
		sortName:  sortName,
		info:      info,
		Cat:       category,
		Title:     title,
		Date:      date,
		url:       url,
		Meta:      fm,
		inGallery: inGallery,
	})
}

func (ss *siteState) loadBlob(file string, info os.FileInfo) {
	_, _, _, _, _, dst := ss.getOutInfo(file, ss.cfg.ContentDir, false)
	ss.addBlob(file, dst)
}

func (ss *siteState) loadPublic(file string, info os.FileInfo) {
	ss.addPublic(file)
}

func (ss *siteState) getOutInfo(file, dir string, isPage bool) (
	cat string,
	date time.Time,
	name, sortName, url, dst string) {

	name = afs.DropRoot(ss.baseDir, dir, file)

	if strings.Count(name, "/") == 0 {
		url = "/" + filepath.Clean(name)
		dst = filepath.Join(ss.baseDir, ss.cfg.PublicDir, name)
		return
	}

	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		cat = parts[0]
		sortName = parts[1]
		date, name = ss.parseName(parts[1])
	} else {
		last := parts[len(parts)-2]
		_, ok := strs.ToDate(last)
		if ok {
			date, name = ss.parseName(last)
			cat = filepath.Join(parts[0 : len(parts)-2]...)
		} else {
			last = parts[len(parts)-1]
			date, name = ss.parseName(last)
			cat = filepath.Join(parts[0 : len(parts)-1]...)
		}

		sortName = last
	}

	name = cfs.ChangeExt(name, "")
	if date.IsZero() {
		url = cat
	} else {
		url = filepath.Join(cat, date.Format("2006/01/02"), name)
	}

	dst = filepath.Join(ss.baseDir, ss.cfg.PublicDir, url)

	if isPage {
		dst = filepath.Join(dst, "index.html")
		url += "/"
	} else {
		base := filepath.Base(file)
		dst = filepath.Join(dst, base)
		url = filepath.Join(url, base)
		sortName = filepath.Join(sortName, base)
	}

	url = "/" + url

	return
}

func (ss *siteState) parseName(name string) (time.Time, string) {
	t, ok := strs.ToDate(name)
	if !ok {
		return t, name
	}

	return t, strings.Trim(name[len(strs.DateFormat):], "-")
}
