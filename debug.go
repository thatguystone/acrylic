package acrylic

import (
	"os"

	"github.com/thatguystone/cog"
)

var debug = false

const (
	envDebug = "ACRYLIC_DEBUG"
	envNode  = "NODE_ENV"
)

func init() {
	debug = os.Getenv(envDebug) == "true"
}

func setDebug() {
	err := os.Setenv(envDebug, "true")
	cog.Must(err, "failed to set "+envDebug)

	err = os.Setenv(envNode, "development")
	cog.Must(err, "failed to set "+envNode)
}

func setProduction() {
	err := os.Setenv(envDebug, "false")
	cog.Must(err, "failed to set "+envDebug)

	err = os.Setenv(envNode, "production")
	cog.Must(err, "failed to set "+envNode)
}

func isDebug() bool {
	return debug
}
