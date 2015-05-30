package toner

import (
	"path/filepath"

	"github.com/rainycape/vfs"
)

// Toner represents a single site
type Toner struct {
	cfg Config
	fs  tfs
}

type tfs interface {
	vfs.VFS
}

func newToner(cfg Config, fs vfs.VFS) *Toner {
	return &Toner{
		cfg: cfg,
		fs:  fs,
	}
}

func New(cfg Config) (*Toner, error) {
	if cfg.Root == "" {
		cfg.Root = "."
	}

	abs, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}

	fs, err := vfs.FS(abs)
	if err != nil {
		return nil, err
	}

	return newToner(cfg, fs), nil
}

// Build builds the current site
func (t *Toner) Build() error {
	if err := t.cfg.reload(); err != nil {
		return err
	}

	return newSite(&t.cfg, t.fs).build()
}
