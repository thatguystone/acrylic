package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

var cleanSafeFiles = []string{
	".git",
	".nojekyll",
}

func cmdClean(cfg Config) {
	fs := filepath.Join(cfg.getPublicDir(), "*")
	paths, err := filepath.Glob(fs)
	if err != nil {
		log.Fatal(err)
	}

OUTER:
	for _, path := range paths {
		for _, sf := range cleanSafeFiles {
			// Be conservative with safe files for now
			if strings.Contains(path, sf) {
				continue OUTER
			}
		}

		err = os.RemoveAll(path)
		if err != nil {
			log.Fatal(err)
		}
	}
}
