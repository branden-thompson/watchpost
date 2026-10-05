package temperature

// totals.go — NDFD's rain and snow a day (W18.5, D-168): Forecast mode's
// fallback while Open-Meteo refuses. NDFD has no hourly rain rate to draw in
// radar's colours, only six-hour amounts - the rain liquid-equivalent (qpf)
// and the snowfall - so the day's totals are drawn in their own scale
// (D-184), and three days and a part are all it reaches.

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/units"
)

// NDFDTotalsCredit is the totals' credit line where NDFD drew them.
const NDFDTotalsCredit = "Rain and snow totals: NWS NDFD, each day's six-hour amounts, interpolated"

// Totals is NDFD's rain and snow a day over a lattice: QPF the rain,
// liquid-equivalent - snow's water counted - in mm, and Snow the snowfall in
// cm; by day from today, then point, each six-hour period counted on the
// local date it starts. Missing is NaN - past NDFD's reach - never dry.
type Totals struct {
	Lattice   Lattice
	QPF, Snow [Days][]float64
}

// newTotals is an answer with every value missing.
func newTotals(l Lattice) Totals {
	out := Totals{Lattice: l}
	for k := range Days {
		out.QPF[k], out.Snow[k] = missing(l.Cols*l.Rows), missing(l.Cols*l.Rows)
	}
	return out
}

// Totals asks NDFD once for the lattice's six-hour rain and snow, from the
// period now falls in, and sums each day's.
func (s *NDFD) Totals(ctx context.Context, l Lattice, now time.Time) (Totals, error) {
	if err := noLattice("NDFD totals", l); err != nil {
		return Totals{}, err
	}
	var list []string
	for _, p := range l.Points() {
		list = append(list, ftoa(p.Lat)+","+ftoa(p.Lon))
	}
	q := url.Values{"listLatLon": {strings.Join(list, " ")}, "product": {"time-series"}, "qpf": {"qpf"}, "snow": {"snow"}}
	body, err := s.get.GetText(ctx, s.base+"/xml/sample_products/browser_interface/ndfdXMLclient.php?"+q.Encode(), httpx.TTL(untilNextHour(now)))
	if err != nil {
		return Totals{}, fmt.Errorf("NDFD totals: %w", err)
	}
	var doc dwml
	if err := xml.Unmarshal(body, &doc); err != nil {
		return Totals{}, fmt.Errorf("NDFD totals: %w", err)
	}
	if len(doc.Data.Locations) == 0 {
		return Totals{}, errors.New("NDFD totals: the answer names no point")
	}
	out := newTotals(l)
	index, layouts := doc.pointIndex(l), doc.layouts(now)
	for _, p := range doc.Data.Parameters { // the lattice's points (P10-02)
		at, ok := index[p.Location]
		if !ok {
			continue
		}
		for _, series := range p.Precip { // rain and snow
			day, scale, b := &out.QPF, units.MmPerInch, precipBound // inches to mm
			if series.Type == "snow" {
				day, scale, b = &out.Snow, 2.54, snowBound // inches to cm
			} else if series.Type != "liquid" {
				continue
			}
			eachValue(series, layouts[series.Layout].starts, func(t time.Time, v float64) {
				if k := dayOffset(t, now); k >= 0 && k < Days && b.holds(v*scale) {
					day[k][at] = sumInto(day[k][at], v*scale)
				}
			})
		}
	}
	return out, nil
}

// sumInto adds an amount to a day's total, missing until the first.
func sumInto(total, v float64) float64 {
	if math.IsNaN(total) {
		return v
	}
	return total + v
}
