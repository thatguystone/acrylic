package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/thatguystone/acrylic/acryliclib"
)

func cmdBuild(cfg Config) error {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	err = os.Chdir(filepath.Dir(cfg.path))
	if err != nil {
		panic(err)
	}

	stats, errs := acryliclib.Build(cfg.Config)
	if len(errs) > 0 {
		return errors.New(errs.String())
	}

	if stats.Duration > time.Millisecond {
		stats.Duration /= time.Millisecond
		stats.Duration *= time.Millisecond
	}

	log.Printf("Site built!")
	log.Printf("    Pages: %d", stats.Pages)
	log.Printf("    JS:    %d", stats.JS)
	log.Printf("    CSS:   %d", stats.CSS)
	log.Printf("    Imgs:  %d", stats.Imgs)
	log.Printf("    Blobs: %d", stats.Blobs)
	log.Printf("    Took:  %v", stats.Duration)

	err = os.Chdir(cwd)
	if err != nil {
		panic(err)
	}

	return nil
}
