package radar

// rate.go — a model's rain in radar's scale (0.18.0 D-115, D-116): where no
// radar forecast is, Open-Meteo's hourly rain is drawn in radar's colours.

import "math"

// NoEcho is a dry hour in radar's scale: IEM's table's floor, below every
// class the map draws.
const NoEcho = -32.0

// DBZOfRate is a rain rate in mm an hour as the reflectivity that would show
// it, by Marshall and Palmer's relation, Z = 200 R^1.6 - the one radar
// itself reads rain rate from. A dry hour is NoEcho; NaN stays NaN.
func DBZOfRate(mmPerHour float64) float64 {
	switch {
	case math.IsNaN(mmPerHour):
		return math.NaN()
	case mmPerHour <= 0:
		return NoEcho
	}
	return max(10*math.Log10(200*math.Pow(mmPerHour, 1.6)), NoEcho)
}
