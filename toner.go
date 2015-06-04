package toner

// Toner represents a single site
type Toner struct {
	cfg Config
}

// BuildStats contains interesting information about a single build of the
// site.
type BuildStats struct {
}

func New(cfg Config) *Toner {
	cfg.setDefaults()

	return &Toner{
		cfg: cfg,
	}
}

// Build builds the current site
func (t *Toner) Build() (BuildStats, Errors) {
	return newSite(&t.cfg).build()
}
