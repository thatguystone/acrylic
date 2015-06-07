package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/thatguystone/acrylic/acrylib"
	"gopkg.in/yaml.v2"
)

// Config adds command configuration options to acrylic's build options.
type Config struct {
	acrylib.Config

	Server struct {
		ListenAddr string `yaml:"listenAddr"`
		NoWatch    bool
	}

	path      string
	hideStats bool
}

func loadConfig(cfgFile string) (cfg Config, err error) {
	cfgb, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		err = fmt.Errorf("config error: %v", err)
		return
	}

	// yaml doesn't seem to like the embedded struct
	err = yaml.Unmarshal(cfgb, &cfg.Config)
	if err != nil {
		err = fmt.Errorf("config error: %v", err)
		return
	}

	err = yaml.Unmarshal(cfgb, &cfg)
	if err != nil {
		err = fmt.Errorf("config error: %v", err)
		return
	}

	cfg.path = cfgFile

	if cfg.Server.ListenAddr == "" {
		cfg.Server.ListenAddr = ":9090"
	}

	return
}

func mustLoadConfig(c *cli.Context) (cfg Config) {
	cfg, err := loadConfig(c.GlobalString("config"))
	if err != nil {
		log.Fatal(err)
	}

	return
}

func (cfg *Config) getPublicDir() string {
	return filepath.Join(filepath.Dir(cfg.path), cfg.PublicDir)
}
