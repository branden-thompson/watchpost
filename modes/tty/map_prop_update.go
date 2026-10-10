package tty

import (
	"context"
	"errors"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// PropagationResult is one Propagation update's snapshot, or why there is
// none (0.19.0 W4.1).
type PropagationResult struct {
	Snapshot ionomaps.Snapshot
	Err      error
}

// propRefreshEvery is how often an open Propagation mode asks again: each
// new GloTEC grid (FR-4.10, D-94's default; the Setting comes later).
const propRefreshEvery = 10 * time.Minute

// propUpdatedMsg is an update's result, to the session of the mode that
// asked; propTickMsg is the refresh tick, to the same.
type (
	propUpdatedMsg struct {
		gen uint64
		res PropagationResult
	}
	propTickMsg struct{ gen uint64 }
)

// askPropagation starts the mode's update, or joins the one running (D-131):
// only in the Propagation mode, with the map the window shown and its
// acknowledgement closed (FR-4.2, D-81), and one at a time. The seam is
// called in the command, never on the UI goroutine (FR-4.8).
func (d Dashboard) askPropagation() (Dashboard, tea.Cmd) {
	seam := d.cfg.PropagationUpdate
	ready := seam != nil && !d.mapPane.propBusy && d.modal == modalMap
	if !ready || d.mapMode() == modeRadar || d.mapMode() == modeForecast {
		return d, nil
	}
	if d.mapPane.propCtx == nil {
		d.mapPane.propCtx, d.mapPane.propCancel = context.WithCancel(context.Background())
	}
	d.mapPane.propBusy = true
	d.mapPane.gen++
	ctx, gen := d.mapPane.propCtx, d.mapPane.propGen
	return d, func() tea.Msg { return propUpdatedMsg{gen: gen, res: <-seam(ctx)} }
}

// stopPropagation ends the mode's session: its update cancelled, a late
// result or tick dropped. The last snapshot is kept to show at once when the
// mode opens again.
func (d Dashboard) stopPropagation() Dashboard {
	if d.mapPane.propCancel != nil {
		d.mapPane.propCancel()
	}
	d.mapPane.propCtx, d.mapPane.propCancel, d.mapPane.propBusy = nil, nil, false
	d.mapPane.propGen++
	return d
}

// applyPropUpdated keeps an update's result and sets the next refresh tick.
func (d Dashboard) applyPropUpdated(v propUpdatedMsg) (tea.Model, tea.Cmd) {
	if v.gen != d.mapPane.propGen {
		return d, nil // a left session's
	}
	d.mapPane.propBusy = false
	d.mapPane.gen++
	if len(v.res.Snapshot.Hours) > 0 {
		snap := v.res.Snapshot
		d.mapPane.propSnap = &snap
	}
	d.mapPane.propErr = v.res.Err
	gen := v.gen
	return d, tea.Tick(propRefreshEvery, func(time.Time) tea.Msg { return propTickMsg{gen: gen} })
}

// applyPropTick asks again on the refresh tick; with another window over the
// map it waits for the next.
func (d Dashboard) applyPropTick(v propTickMsg) (tea.Model, tea.Cmd) {
	if v.gen != d.mapPane.propGen {
		return d, nil
	}
	nd, cmd := d.askPropagation()
	if cmd != nil {
		return nd, cmd
	}
	gen := v.gen
	return d, tea.Tick(propRefreshEvery, func(time.Time) tea.Msg { return propTickMsg{gen: gen} })
}

// propStatusWords are the Propagation mode's status: asking, what the last
// snapshot was made from and when, or why there is none - the library's
// words, which say when it asks again (D-124).
func (d Dashboard) propStatusWords() string {
	snap, err := d.mapPane.propSnap, d.mapPane.propErr
	switch {
	case snap != nil:
		return snapshotWords(*snap)
	case err != nil && !errors.Is(err, context.Canceled):
		return "Propagation · " + strings.TrimPrefix(err.Error(), "ionomaps: ")
	case d.mapPane.propBusy:
		return "Propagation · asking NOAA for the newest grid"
	}
	return propagationStatus
}

// snapshotWords say what a snapshot was made from and when (FR-5.1).
func snapshotWords(s ionomaps.Snapshot) string {
	words := []string{"Propagation"}
	switch s.Background.FoF2 {
	case ionomaps.GloTEC:
		words = append(words, "foF2 from GloTEC at "+s.Inputs.GloTECValid.UTC().Format("15:04 UTC"))
	case ionomaps.Climatology:
		words = append(words, "foF2 from the climatology: no GloTEC grid")
	}
	words = append(words, "M(3000)F2 from the "+s.Background.M3000.String(), "computed "+s.Computed.UTC().Format("15:04 UTC"))
	switch s.Early {
	case ionomaps.TooSoon:
		words = append(words, "asked too soon: the last field")
	case ionomaps.Offline:
		words = append(words, "NOAA not reached: the last field")
	}
	return strings.Join(words, " · ")
}
