package toner

type Config struct {
	Root  string // Where the site files live, relative to current directory
	Theme string // Name of the theme to use
}

func (cfg *Config) reload() error {
	return nil
}
