package geo

import "strings"

// ParseLocator reads a Maidenhead locator of four, six or eight characters
// ("DM13", "DM13af", "DM13af45"), either case: the centre of its square;
// false for anything else (FR-1.10, D-79).
//
// Source: the Maidenhead Locator System - a field of 20° by 10° (A to R),
// a square of 2° by 1° (0 to 9), a subsquare of 5' by 2.5' (A to X), an
// extended square a tenth of that each way (0 to 9).
func ParseLocator(s string) (LatLon, bool) {
	u := strings.ToUpper(strings.TrimSpace(s))
	n := len(u)
	if n != 4 && n != 6 && n != 8 {
		return LatLon{}, false
	}
	if !in(u[0], 'A', 'R') || !in(u[1], 'A', 'R') || !in(u[2], '0', '9') || !in(u[3], '0', '9') {
		return LatLon{}, false
	}
	lon := float64(u[0]-'A')*20 - 180 + float64(u[2]-'0')*2
	lat := float64(u[1]-'A')*10 - 90 + float64(u[3]-'0')
	w, h := 2.0, 1.0
	if n >= 6 {
		if !in(u[4], 'A', 'X') || !in(u[5], 'A', 'X') {
			return LatLon{}, false
		}
		w, h = w/24, h/24
		lon, lat = lon+float64(u[4]-'A')*w, lat+float64(u[5]-'A')*h
	}
	if n == 8 {
		if !in(u[6], '0', '9') || !in(u[7], '0', '9') {
			return LatLon{}, false
		}
		w, h = w/10, h/10
		lon, lat = lon+float64(u[6]-'0')*w, lat+float64(u[7]-'0')*h
	}
	return LatLon{Lat: lat + h/2, Lon: lon + w/2}, true
}

// LocatorLabel is a locator as it is written: the field and square in
// capitals, the subsquare in small letters ("DM13af").
func LocatorLabel(s string) string {
	u := strings.ToUpper(strings.TrimSpace(s))
	if len(u) >= 6 {
		return u[:4] + strings.ToLower(u[4:6]) + u[6:]
	}
	return u
}

// in reports whether a byte is in a range.
func in(b, lo, hi byte) bool { return b >= lo && b <= hi }
