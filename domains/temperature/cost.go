package temperature

// cost.go — what an ask costs Open-Meteo's quota (W18.6, D-185): it bills
// every location of a request, not the request. Its maintainer (open-meteo
// issues #438 and #1295): a location's weight is its variables over ten
// times its fortnights, at least one, and the request's that times its
// locations - the pricing page's "15 weather variables ... 1.5 API calls,
// while 4 weeks of data equals 3.0".

import (
	"math"
	"net/url"
	"strconv"
	"strings"
)

// CallWeight is an ask's weight against Open-Meteo's quota: none for a host
// that is not Open-Meteo's, or an address that does not parse.
func CallWeight(rawURL string) float64 {
	u, err := url.Parse(rawURL)
	if err != nil || !strings.HasSuffix(u.Hostname(), "open-meteo.com") {
		return 0
	}
	q := u.Query()
	locations := len(listOf(q.Get("latitude")))
	if locations == 0 {
		return 0
	}
	variables := 0
	for _, key := range []string{"hourly", "daily", "current", "minutely_15"} {
		variables += len(listOf(q.Get(key)))
	}
	one := float64(variables) / 10 * math.Max(1, daysAsked(q)/14)
	return float64(locations) * math.Max(1, one)
}

// listOf is a comma list's items, none for an empty one.
func listOf(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// daysAsked is the span an ask covers, in days: its days, or its hours.
func daysAsked(q url.Values) float64 {
	n := func(key string) float64 {
		v, _ := strconv.ParseFloat(q.Get(key), 64)
		return v
	}
	days := n("forecast_days") + n("past_days")
	if hours := (n("forecast_hours") + n("past_hours")) / 24; hours > days {
		return hours
	}
	return days
}
