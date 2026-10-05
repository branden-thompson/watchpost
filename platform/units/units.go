// Package units is every unit conversion the app makes, in one place (W14
// S-6): temperature, distance and speed, each by its definition. A leaf -
// it imports nothing - so domains, platform and modes alike read the same
// constants, and the hazard tape and the bed fence measure a mile the same.
package units

// KmPerMile is the international mile in kilometres, exactly.
const KmPerMile = 1.609344

// KmhPerMps is a metre a second in kilometres an hour, exactly.
const KmhPerMps = 3.6

// KmhPerKnot is a knot in kilometres an hour, exactly.
const KmhPerKnot = 1.852

// FahrenheitOf is a Celsius temperature in Fahrenheit.
func FahrenheitOf(c float64) float64 { return c*9/5 + 32 }

// CelsiusOf is a Fahrenheit temperature in Celsius.
func CelsiusOf(f float64) float64 { return (f - 32) * 5 / 9 }

// FahrenheitDelta is a Celsius DIFFERENCE in Fahrenheit degrees: scaled, with
// no offset - feels-like 5 °C above the air is 9 °F above it, not 41.
func FahrenheitDelta(dc float64) float64 { return dc * 9 / 5 }

// MilesOf is kilometres in miles - and km/h in mph.
func MilesOf(km float64) float64 { return km / KmPerMile }

// KmOf is miles in kilometres - and mph in km/h.
func KmOf(mi float64) float64 { return mi * KmPerMile }

// KmhOf is metres a second in kilometres an hour.
func KmhOf(mps float64) float64 { return mps * KmhPerMps }

// MpsOf is kilometres an hour in metres a second.
func MpsOf(kmh float64) float64 { return kmh / KmhPerMps }

// KmhOfKnots is knots in kilometres an hour.
func KmhOfKnots(kt float64) float64 { return kt * KmhPerKnot }

// MetresPerFoot is the international foot in metres, exactly.
const MetresPerFoot = 0.3048

// MmPerInch is the inch in millimetres, exactly.
const MmPerInch = 25.4

// FeetOf is metres in feet.
func FeetOf(m float64) float64 { return m / MetresPerFoot }

// MetresOfFeet is feet in metres.
func MetresOfFeet(ft float64) float64 { return ft * MetresPerFoot }

// InchesOf is millimetres in inches.
func InchesOf(mm float64) float64 { return mm / MmPerInch }

// MmOfInches is inches in millimetres.
func MmOfInches(in float64) float64 { return in * MmPerInch }

// KnotsOf is metres a second in knots.
func KnotsOf(mps float64) float64 { return KmhOf(mps) / KmhPerKnot }

// MphOf is metres a second in miles an hour.
func MphOf(mps float64) float64 { return MilesOf(KmhOf(mps)) }

// MpsOfMph is miles an hour in metres a second.
func MpsOfMph(mph float64) float64 { return MpsOf(KmOf(mph)) }
