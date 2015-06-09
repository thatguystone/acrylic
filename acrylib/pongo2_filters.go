package acrylib

import (
	p2 "github.com/flosch/pongo2"
)

const p2ContentRelPfx = "crel://"

func init() {
	p2.RegisterFilter("content_rel", p2ContentRelFilter)
}

func p2ContentRelFilter(in *p2.Value, param *p2.Value) (out *p2.Value, err *p2.Error) {
	return p2.AsValue(p2ContentRelPfx + in.String()), nil
}
