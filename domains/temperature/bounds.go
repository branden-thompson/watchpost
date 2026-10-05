package temperature

// bounds.go — the range each measure can physically take. Every value a
// source answers is held to its measure's bound where it is read, in the
// measure's own units: a value outside it is broken, and stays missing, as
// NaN and Inf do - never drawn, never recorded.

// bound is a measure's physical range, both ends included.
type bound struct{ lo, hi float64 }

var (
	celsiusBound   = bound{-90, 60} // °C: the air, or how it feels
	windBound      = bound{0, 500}  // km/h: sustained wind or gust
	directionBound = bound{0, 360}  // degrees the wind blows from
	uvBound        = bound{0, 20}   // the UV index
	precipBound    = bound{0, 1000} // mm of rain, or of snow's water
	snowBound      = bound{0, 100}  // cm of snowfall: precipBound's 1000 mm
	waveBound      = bound{0, 30}   // m of significant wave height
)

// holds reports whether a value is a number within the bound; NaN never is.
func (b bound) holds(v float64) bool { return v >= b.lo && v <= b.hi }
