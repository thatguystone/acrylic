package data

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/thatguystone/acrylic/internal/afs"
	"github.com/thatguystone/acrylic/internal/config"
	"github.com/thatguystone/cog/cfs"
)

type D struct {
	cfg *config.C

	rwmtx sync.RWMutex
	ds    map[string]interface{}
}

func New(cfg *config.C) *D {
	return &D{
		cfg: cfg,
		ds:  map[string]interface{}{},
	}
}

func (d *D) LoadData(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	var data []byte

	cached := filepath.Join(d.cfg.CacheDir, "data", info.Name())

	exists, _ := cfs.FileExists(cached)
	if exists && !afs.SrcChanged(path, cached) {
		data, err = ioutil.ReadFile(cached)
		if err != nil {
			return err
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
			cmd := exec.Command(path)

			ob := bytes.Buffer{}
			cmd.Stdout = &ob

			eb := bytes.Buffer{}
			cmd.Stderr = &eb

			err := cmd.Run()
			if err != nil || eb.Len() > 0 {
				return fmt.Errorf("execute failed: %v: %s",
					err,
					eb.String())
			}

			data = ob.Bytes()
		} else {
			data, err = ioutil.ReadFile(path)
			if err != nil {
				return err
			}
		}
	}

	var v interface{}
	err = json.Unmarshal(data, &v)
	if err != nil {
		return err
	}

	if v, ok := v.(map[string]interface{}); ok {
		if until, ok := v["acrylic_expires"].(float64); ok {
			b := bytes.Buffer{}
			binary.Write(&b, binary.BigEndian, uint64(until))
			b.Write(data)

			err = cfs.Write(cached, b.Bytes())
			if err != nil {
				return fmt.Errorf("failed to write cache file: %v", err)
			}

			os.Chtimes(cached, info.ModTime(), info.ModTime())
		}

		delete(v, "acrylic_expires")
	}

	d.rwmtx.Lock()
	d.ds[cfs.DropExt(info.Name())] = v
	d.rwmtx.Unlock()

	return nil
}

func (d *D) Get(name string) (di interface{}) {
	d.rwmtx.RLock()
	di = d.ds[name]
	d.rwmtx.RUnlock()

	return
}
