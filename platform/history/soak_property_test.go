//go:build property

package history

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// NINETY DAYS OF A GLOBAL GRID STAY READABLE (#27, 0.19.0 W1.2): the largest
// dataset 0.19.0 records - two fields over the world at 2°, 180 x 91 points -
// kept ninety days at the real budget. Every part stays within the budget,
// so within the read cap, and every day is read back. Synthetic values, two
// decimals as MUF keeps them; three records a day, since a roll-up's size is
// the grid's, not the hours'. A 31-day month of it is about 16.7 MB, just
// within one part's budget; a further field splits it. It writes about half a
// gigabyte, so it runs once, in `make property`.
func TestNinetyDaysOfAGlobalGridStayReadable(t *testing.T) {
	const cols, rows, days = 180, 91, 90
	world := Dataset{Name: "world", Version: 1, Hours: 72 * time.Hour, Days: 400 * 24 * time.Hour,
		Fields: []Field{{Name: "muf", Unit: "MHz", Decimals: 2}, {Name: "fof2", Unit: "MHz", Decimals: 2}}}
	shape := Shape{Box: geo.Box{W: -180, S: -90, E: 180, N: 90}, Cols: cols, Rows: rows}
	key := Key{Source: "synthetic", Place: "world"}
	dir := t.TempDir()
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	now := start
	s := Open(dir, func() time.Time { return now }, world)
	field := func(at time.Time, scale float64) []float64 {
		v := make([]float64, cols*rows)
		hour := float64(at.Unix()/3600) / 24
		for i := range v {
			v[i] = scale * (8 + 6*math.Sin(float64(i)/97+hour) + 3*math.Cos(float64(i)/13-hour))
		}
		return v
	}
	for d := 0; d < days+4; d++ { // the days, then retention's few past them
		day := start.AddDate(0, 0, d)
		for h := 0; d < days && h < 24; h += 8 {
			at := day.Add(time.Duration(h) * time.Hour)
			r := Record{Key: key, At: at, IssuedAt: at, Shape: shape,
				Values: map[string][]float64{"muf": field(at, 2.2), "fof2": field(at, 1)}}
			if !s.Put(world.Name, r) {
				t.Fatalf("day %d hour %d was not written", d, h)
			}
		}
		now = day.Add(24 * time.Hour)
		for i := 0; i < 8; i++ {
			s.RollUpAndPrune()
		}
	}
	got := s.Days(world.Name, key, start, start.AddDate(0, 0, days-1), days+1)
	if len(got) != days {
		t.Fatalf("read back %d days of %d", len(got), days)
	}
	parts := 0
	root := filepath.Join(dir, world.Name, "v1", key.Source, key.Place, "rollup")
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".json.gz") {
			return err
		}
		parts++
		n := decompressedSize(t, p)
		t.Logf("%s: %d bytes decompressed, %d on disk", filepath.Base(p), n, info.Size())
		if n > int64(rollUpBudget) {
			t.Errorf("%s decompresses to %d bytes, past the budget %d", filepath.Base(p), n, rollUpBudget)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if parts < 3 {
		t.Errorf("ninety days over three months in %d part(s): a part holds one month's days at most", parts)
	}
}
