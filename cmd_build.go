package main

import (
	"errors"
	"fmt"

	"github.com/thatguystone/acrylic/acryliclib"
)

func cmdBuild(cfgFile string) error {
	stats, errs := acryliclib.Build(acryliclib.Config{})
	if len(errs) > 0 {
		return errors.New(errs.String())
	}

	fmt.Println("Site built!")
	fmt.Printf("	Pages: %d\n", stats.Pages)
	fmt.Printf("	JS:    %d\n", stats.JS)
	fmt.Printf("	CSS:   %d\n", stats.CSS)
	fmt.Printf("	Imgs:  %d\n", stats.Imgs)
	fmt.Printf("	Blobs: %d\n", stats.Blobs)
	fmt.Printf("	Took:  %v\n", stats.Duration)

	return nil
}
