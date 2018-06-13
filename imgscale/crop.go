package imgscale

import (
	"encoding"
	"fmt"
	"strings"
)

type cropGravity int

const (
	center cropGravity = iota // Center is the default
	northWest
	north
	northEast
	west
	east
	southWest
	south
	southEast
)

var (
	_ encoding.TextMarshaler   = cropGravity(0)
	_ encoding.TextUnmarshaler = (*cropGravity)(nil)
	_ fmt.Stringer             = cropGravity(0)
)

func (g cropGravity) MarshalText() ([]byte, error) {
	return []byte(g.shortName()), nil
}

func (g *cropGravity) UnmarshalText(b []byte) error {
	switch strings.ToLower(string(b)) {
	case "c", "center":
		*g = center
	case "nw", "northwest":
		*g = northWest
	case "n", "north":
		*g = north
	case "ne", "northeast":
		*g = northEast
	case "w", "west":
		*g = west
	case "e", "east":
		*g = east
	case "sw", "southwest":
		*g = southWest
	case "s", "south":
		*g = south
	case "se", "southeast":
		*g = southEast
	default:
		return fmt.Errorf("unrecognized gravity: %q", string(b))
	}

	return nil
}

func (g cropGravity) shortName() string {
	switch g {
	case center:
		return "c"
	case northWest:
		return "nw"
	case north:
		return "n"
	case northEast:
		return "ne"
	case west:
		return "w"
	case east:
		return "e"
	case southWest:
		return "sw"
	case south:
		return "s"
	case southEast:
		return "se"
	default:
		panic(fmt.Errorf("unrecognized gravity: %d", g))
	}
}

func (g cropGravity) String() string {
	switch g {
	case center:
		return "Center"
	case northWest:
		return "NorthWest"
	case north:
		return "North"
	case northEast:
		return "NorthEast"
	case west:
		return "West"
	case east:
		return "East"
	case southWest:
		return "SouthWest"
	case south:
		return "South"
	case southEast:
		return "SouthEast"
	default:
		panic(fmt.Errorf("unrecognized gravity: %d", g))
	}
}
