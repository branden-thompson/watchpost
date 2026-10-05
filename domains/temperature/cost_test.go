package temperature

import (
	"math"
	"testing"
)

// AN ASK WEIGHS AS OPEN-METEO BILLS IT (W18.6, D-185): one call a location,
// times a tenth of its variables past ten, times its fortnights past one -
// its maintainer's formula (open-meteo issues #438, #1295) and the pricing
// page's examples - and a host that is not Open-Meteo's weighs nothing.
func TestAnAskWeighsAsOpenMeteoBillsIt(t *testing.T) {
	for _, c := range []struct {
		url  string
		want float64
	}{
		{"https://api.open-meteo.com/v1/forecast?latitude=1&longitude=2&hourly=temperature_2m&forecast_days=7", 1},
		{"https://api.open-meteo.com/v1/forecast?latitude=1,2,3&longitude=4,5,6&hourly=a,b,c,d,e,f&daily=g,h,i,j,k,l,m,n&forecast_days=7", 3 * 1.4},
		{"https://marine-api.open-meteo.com/v1/marine?latitude=1,2&longitude=3,4&hourly=wave_height&daily=wave_height_max&past_hours=3&forecast_hours=13&forecast_days=7", 2},
		{"https://api.open-meteo.com/v1/forecast?latitude=1&longitude=2&hourly=a,b,c,d,e,f,g,h,i,j,k,l,m,n,o&forecast_days=14", 1.5},
		{"https://api.open-meteo.com/v1/forecast?latitude=1&longitude=2&hourly=a,b,c,d,e,f,g,h,i,j,k,l,m,n,o&forecast_days=28", 3},
		{"https://graphical.weather.gov/xml/sample_products/browser_interface/ndfdXMLclient.php?listLatLon=1,2", 0},
		{"not a url %%", 0},
	} {
		if got := CallWeight(c.url); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%s weighs %v; want %v", c.url, got, c.want)
		}
	}
}
