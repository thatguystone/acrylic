package acryliclib

// Acrylic represents a single site
type Acrylic struct {
	cfg Config
}

// BuildStats contains interesting information about a single build of the
// site.
type BuildStats struct {
}

func New(cfg Config) *Acrylic {
	cfg.setDefaults()

	return &Acrylic{
		cfg: cfg,
	}
}

// Build builds the current site
func (t *Acrylic) Build() (BuildStats, Errors) {
	return newSite(&t.cfg).build()
}
