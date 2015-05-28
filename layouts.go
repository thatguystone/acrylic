package toner

import "fmt"

type layouts struct {
	t *Toner
	d data
}

func newLayouts(t *Toner, d data) (*layouts, error) {
	l := &layouts{
		t: t,
		d: d,
	}

	if l.t.cfg.Theme == "" {
		return l, nil
	}

	base := fmt.Sprintf("/themes/%s/base.html", l.t.cfg.Theme)
	if !l.t.fexists(base) {
		return nil, fmt.Errorf("theme `%s` does not exist",
			l.t.cfg.Theme)
	}

	return l, nil
}
