package app

// maps.go — the Observer's map, built at the composition root (0.18.0).
//
// THE MAP IS BUILT HERE AND HANDED TO THE WINDOW, which builds it on the
// listener's first g (FR-3.2): nothing is constructed, and nothing is reached,
// before the listener asks for a map. The map names no source yet: the
// library's embedded tiles draw the basemap, offline. The network basemap
// source is W3.1, and whether maps are on at all is W1.8's Setting.

import (
	tuimaps "github.com/branden-thompson/go-tuimaps"
	"github.com/branden-thompson/go-tuimaps/assets"
)

// newMap builds the map at its window's size: the library moves only a map
// that has a size, so the window's first view can be placed at once.
func newMap(size tuimaps.Size) (*tuimaps.Map, error) {
	return tuimaps.New(tuimaps.WithSize(size.Cols, size.Rows), tuimaps.Embed(assets.Tile, assets.MaxZoom))
}
