package radar

import (
	"bytes"
	"errors"
	"image/png"
)

// ErrTooLarge is a picture whose header declares more pixels than the
// library takes - its image cap, one byte a pixel: refused before anything
// decodes (W8.5).
var ErrTooLarge = errors.New("radar: the picture declares more pixels than a frame may have")

// Check reads a frame's header before its pixels, refusing a picture too
// large to decode, and reports whether it paints anything. An empty picture
// at an advertised time is a real frame with no echo (D-84).
func Check(frame []byte) (empty bool, err error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(frame))
	if err != nil {
		return false, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
		return false, ErrTooLarge
	}
	img, err := png.Decode(bytes.NewReader(frame))
	if err != nil {
		return false, err
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				return false, nil
			}
		}
	}
	return true, nil
}
