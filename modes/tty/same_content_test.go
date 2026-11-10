package tty

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// republished is a fresh snapshot with s's locations, as a publish that
// changed none of them makes it: a new pointer, a later stamp, the providers
// fetched again. edit changes the copy's locations.
//
// The locations are copied deeply, as the assembler builds each publish
// afresh. A shallow copy would share their maps and slices, and
// reflect.DeepEqual answers a shared map without reading it, so the
// comparison D-31 adds would look cheaper than it is.
func republished(tb testing.TB, s *snapshot.Snapshot, edit func([]snapshot.Location) []snapshot.Location) *snapshot.Snapshot {
	tb.Helper()
	next := *s
	next.GeneratedAt = s.GeneratedAt.Add(20 * time.Second)
	next.Providers = append([]snapshot.ProviderStatus(nil), s.Providers...)
	for i := range next.Providers {
		next.Providers[i].FetchedAt = next.Providers[i].FetchedAt.Add(20 * time.Second)
	}
	next.Locations = deepLocations(tb, s.Locations)
	if edit != nil {
		next.Locations = edit(next.Locations)
	}
	return &next
}

// deepLocations is ls copied through gob: equal to it, and sharing none of its
// maps, slices or pointers. gob reads a nil slice back as nil and an empty one
// as nil too, so a fixture with an empty slice would come back unequal; the
// check says so rather than letting a pin measure a redraw.
func deepLocations(tb testing.TB, ls []snapshot.Location) []snapshot.Location {
	tb.Helper()
	var buf bytes.Buffer
	var out []snapshot.Location
	if err := gob.NewEncoder(&buf).Encode(ls); err != nil {
		tb.Fatal(err)
	}
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		tb.Fatal(err)
	}
	if !reflect.DeepEqual(out, ls) {
		tb.Fatal("the copied locations differ from the originals, so a publish of them is not a no-change publish")
	}
	return out
}

// TestANoChangePublishRedrawsNoTable is 0.19.5 D-31 (PF-9): a publish whose
// locations are the ones already drawn changes nothing the tables show, so
// the frame after it draws them from the memo, for the watchlist and for
// RECENT alike.
func TestANoChangePublishRedrawsNoTable(t *testing.T) {
	d := benchDash(t, 133, 44).(Dashboard)
	_ = d.View().Content
	_, misses := d.memoCounts()
	for range 3 { // three publishes in a row, each repeating the one before (P10-02)
		m, _ := d.Update(SnapshotMsg{Snap: republished(t, d.snap, nil)})
		m, _ = m.(Dashboard).Update(RecentSnapshotMsg{Snap: republished(t, m.(Dashboard).recent, nil)})
		d = m.(Dashboard)
		_ = d.View().Content
	}
	if _, now := d.memoCounts(); now != misses {
		t.Errorf("three publishes that changed no location redrew the tables %d time(s)", now-misses)
	}
}

// TestAChangedPublishRedrawsTheTables is D-31's other half, the stale-row
// cases of issues #11 and #12: a changed reading, a place removed and a place
// renamed each reach the table's row in the frame after the publish. The
// row is looked for by its number, since the radio's station line names the
// focused place as well and is not memoised.
func TestAChangedPublishRedrawsTheTables(t *testing.T) {
	for _, c := range []struct {
		name  string
		edit  func([]snapshot.Location) []snapshot.Location
		shows func(string) bool
	}{
		{"a changed reading", func(l []snapshot.Location) []snapshot.Location {
			v := 41.0
			l[0].Harmonized.Temp = &v
			return l
		}, func(f string) bool { return strings.Contains(f, "106°F") }},
		{"a place removed (#11)", func(l []snapshot.Location) []snapshot.Location { return l[1:] },
			func(f string) bool { return !strings.Contains(f, "001. Benchmark City 01 ") }},
		{"a place renamed (#12)", func(l []snapshot.Location) []snapshot.Location {
			l[0].Label = "Grandma's House"
			return l
		}, func(f string) bool { return strings.Contains(f, "001. Grandma's House") }},
	} {
		d := benchDash(t, 133, 44).(Dashboard)
		_ = d.View().Content
		m, _ := d.Update(SnapshotMsg{Snap: republished(t, d.snap, c.edit)})
		if frame := stripANSITest(m.View().Content); !c.shows(frame) {
			t.Errorf("%s did not reach the table:\n%s", c.name, frame)
		}
	}
}

// TestAWrittenSnapshotStandsForItself is D-31's safety: only a snapshot that
// arrived through applySnapshot is compared with the one before it, so one
// written straight onto the model, as a test fixture does, is drawn as it is
// and never as the last snapshot that arrived.
func TestAWrittenSnapshotStandsForItself(t *testing.T) {
	d := benchDash(t, 133, 44).(Dashboard)
	m, _ := d.Update(SnapshotMsg{Snap: republished(t, d.snap, nil)})
	d = m.(Dashboard)
	_ = d.View().Content
	d.snap = republished(t, d.snap, func(l []snapshot.Location) []snapshot.Location {
		l[0].Label = "Written Straight On"
		return l
	})
	if frame := stripANSITest(d.View().Content); !strings.Contains(frame, "001. Written Straight On") { // the table's row: the radio's station line names the focus too
		t.Errorf("a snapshot written onto the model drew the one that last arrived:\n%s", frame)
	}
}
