package main

import "io/ioutil"

const defaultConfig = `# Acrylic Configuration
`

func cmdNew(cfgFile string) error {
	err := ioutil.WriteFile(cfgFile, []byte(defaultConfig), 0640)
	if err != nil {
		return err
	}

	return nil
}
