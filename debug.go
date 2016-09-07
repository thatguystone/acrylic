package acrylic

import (
	"os"

	"github.com/thatguystone/cog"
)

var debug = false

const envDebug = "ACRYLIC_DEBUG"

func init() {
	debug = os.Getenv(envDebug) == "true"
}

func setDebug() {
	err := os.Setenv(envDebug, "true")
	cog.Must(err, "failed to set "+envDebug)
}

func isDebug() bool {
	return debug
}
