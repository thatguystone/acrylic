package acrylic

import "testing"

func TestWebpack(t *testing.T) {
	wp := Webpack{}
	wp.Start()
	select {}
}
