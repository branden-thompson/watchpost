package tty

import (
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// The night's roles on the map (go-tuiMaps P4): the night side's shade and
// its dotted edge, the terminator.
const (
	terminatorRole = tuimaps.TerminatorLine
	nightRole      = tuimaps.NightTint
)

// nightStep is how far the night drawn may lag the moment shown: the line
// moves about a quarter of a degree a minute.
const nightStep = 5 * time.Minute

// nightDetailKey is Day / night's key among the map's detail (D-199), and
// mapNightID its overlay's.
const (
	nightDetailKey = "night"
	mapNightID     = "map/night"
)

// mapNight is the night side and its edge at a moment (L-4.1 to L-4.3): no
// words of its own, the edge dotted at every depth - go-tuiMaps' own
// composition of the two, in its roles.
func mapNight(at time.Time) tuimaps.Overlay { return tuimaps.NightOverlay(mapNightID, at) }

// nightWanted is whether the night is drawn and for what moment (D-174,
// D-199): in the Propagation mode always once it has a snapshot, for the hour
// shown; in the weather modes while Day / night is on - Forecast mode's for
// the middle of its step, Radar mode's for the clock.
func (d Dashboard) nightWanted() (time.Time, bool) {
	switch {
	case d.inPropagation():
		return d.propNightMoment(), d.mapPane.propSnap != nil
	case !d.detailOn(nightDetailKey):
		return time.Time{}, false
	case d.mapMode() == modeForecast:
		steps := d.forecastSteps()
		if len(steps) == 0 {
			return d.now().UTC().Truncate(nightStep), true
		}
		sp := steps[min(d.mapPane.fcStep, len(steps)-1)].Span
		return sp.From.Add(sp.Until.Sub(sp.From) / 2).UTC().Truncate(nightStep), true
	}
	return d.now().UTC().Truncate(nightStep), true
}

// setNight hands the library the night as wanted, or takes it off.
func (d Dashboard) setNight() Dashboard {
	if d.mapPane.m == nil {
		return d
	}
	var want []tuimaps.Overlay
	at, on := d.nightWanted()
	if on {
		want = []tuimaps.Overlay{mapNight(at)}
	}
	d.mapPane.nightGiven, _ = d.reconcile(d.mapPane.nightGiven, want, func(o tuimaps.Overlay, err error) {
		d.problem("Day / night: " + o.ID + " not drawn - " + err.Error()) // ours to fix, never the listener's (D-124)
	})
	d.mapPane.nightAt = at
	return d
}

// followNight hands the night in again when the moment it is for has moved a
// step, or takes it off or on, while the map is shown.
func (d Dashboard) followNight() Dashboard {
	if d.modal != modalMap || d.mapPane.m == nil {
		return d
	}
	at, on := d.nightWanted()
	if on == (len(d.mapPane.nightGiven) > 0) && (!on || at.Equal(d.mapPane.nightAt)) {
		return d
	}
	return d.setNight().renderMap()
}

// nightWords say the night on the map (FR-3.1, D-199): its shade and dotted
// edge - the edge alone in the Propagation mode, whose field the shade does
// not darken (AX-13) - and the moment they are for, by the window's clock
// as every other time in the words (AX-8); none where it is not drawn.
func (d Dashboard) nightWords() string {
	if len(d.mapPane.nightGiven) == 0 {
		return ""
	}
	at := d.mapPane.nightAt.In(d.now().Location())
	when := d.clockFmt.Time(at) + " " + at.Format("MST")
	if d.mapMode() == modePropagation {
		return "The night side's edge is dotted, as at " + when + "."
	}
	return "The night side is shaded and its edge dotted, as at " + when + "."
}
