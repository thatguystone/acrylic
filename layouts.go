package toner

import (
	"fmt"
	"path/filepath"
)

type layouts struct {
	s *site
}

func (l *layouts) check() error {
	if l.s.cfg.Theme == "" {
		return nil
	}

	base := filepath.Join(l.s.cfg.ThemesDir, l.s.cfg.Theme, "base.html")
	if !l.s.fexists(base) {
		return fmt.Errorf("theme `%s` does not exist",
			l.s.cfg.Theme)
	}

	return nil
}
