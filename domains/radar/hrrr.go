package radar

// hrrr.go — the radar loop's hours ahead (0.18.0 D-113, D-114): NCEP's HRRR
// forecast reflectivity as IEM draws it, on the host already on FR-3.8's
// list. A run's frames are its minutes ahead of its start, which IEM states
// beside them; each is asked for at that minute, never a time it does not
// hold. The lower 48 only - HRRR's own grid.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// HRRRStep is the forecast frames' step: HRRR's quarter-hours, which run to
// eighteen hours ahead - past every horizon offered (D-114).
const HRRRStep = 15 * time.Minute

// hrrrLastQuarter is the last minute HRRR states on quarter-hours.
const hrrrLastQuarter = 18 * 60

// HRRR is the forecast reflectivity source.
type HRRR struct {
	get  Getter
	base string
}

// NewHRRR builds it; base "" is IEM's host.
func NewHRRR(get Getter, base string) *HRRR {
	if base == "" {
		base = iemBase
	}
	return &HRRR{get: get, base: base}
}

func (s *HRRR) Name() string { return "HRRR" }

// Covers is the lower 48 alone.
func (s *HRRR) Covers(region string) bool { return region == geo.RegionContiguous }

// Run is the newest run's start, as IEM states it.
func (s *HRRR) Run(ctx context.Context) (time.Time, error) {
	body, err := s.get.GetText(ctx, s.base+"/data/gis/images/4326/hrrr/refd_0000.json", httpx.TTL(5*time.Minute))
	if err != nil {
		return time.Time{}, fmt.Errorf("HRRR run: %w", err)
	}
	var meta struct {
		Init string `json:"model_init_utc"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return time.Time{}, fmt.Errorf("HRRR run: %w", err)
	}
	t, err := time.Parse(time.RFC3339, meta.Init)
	if err != nil {
		return time.Time{}, fmt.Errorf("HRRR run: %w", err)
	}
	return t, nil
}

// Minutes are the run's quarter-hours that fall after one moment and at or
// before another: the forecast frames between the newest observed frame and
// the horizon.
func Minutes(run, after, until time.Time) []int {
	var out []int
	for m := 0; m <= hrrrLastQuarter; m += int(HRRRStep / time.Minute) {
		at := run.Add(time.Duration(m) * time.Minute)
		if at.After(after) && !at.After(until) {
			out = append(out, m)
		}
	}
	return out
}

// Frame is one forecast frame of a box, at a minute of run, at half the
// radar box's size - about 8 km, near HRRR's own 3 km at the scale a box is
// drawn, and a quarter of a radar frame's memory (D-114).
//
// THE RESPONSE IS KEPT PER RUN (W14 P-15). The address names the minute
// alone and the server answers it from its newest run, so the run rides in
// the address's fragment: the response cache keys on the whole address, and
// a fragment is never sent - the server sees the same request for every run,
// and a new run's minute is never answered with the last run's picture.
func (s *HRRR) Frame(ctx context.Context, run time.Time, minute int, b Box) ([]byte, error) {
	if minute < 0 || minute > hrrrLastQuarter || minute%int(HRRRStep/time.Minute) != 0 {
		return nil, fmt.Errorf("HRRR frame: minute %d is not one of the run's quarter-hours", minute)
	}
	if err := b.askable(); err != nil {
		return nil, err
	}
	half := Box{Name: b.Name, W: b.W, S: b.S, E: b.E, N: b.N, Cols: max(b.Cols/2, 1), Rows: max(b.Rows/2, 1)}
	q := wmsQuery(half, "LAYERS", "refd_"+fmt.Sprintf("%04d", minute))
	return s.get.GetText(ctx, s.base+"/cgi-bin/wms/hrrr/refd.cgi?"+q.Encode()+"#run="+run.UTC().Format("2006010215"), httpx.TTL(window))
}
