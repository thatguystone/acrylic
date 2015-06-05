package main

import (
	"io/ioutil"
	"log"
)

const defaultConfig = `# Acrylic Configuration
#
# This file is left almost-blank by default since the application provides
# sensible defaults for everything. For a full list of configuration options
# and values, see https://thatguystone.github.io/acrylic/config/

# Set this to the name of your theme.
#theme =
`

func cmdNew(cfgFile string) {
	err := ioutil.WriteFile(cfgFile, []byte(defaultConfig), 0640)
	if err != nil {
		log.Fatal(err)
	}
}
