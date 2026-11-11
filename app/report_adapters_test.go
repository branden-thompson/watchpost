package app

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestAMarineReportReadsTidesInTheLocationsZone is 0.19.5 Q2.4 (RN-5): a tide
// happens where the water is, so its times are read in the location's zone
// (MVS-D-21); a zone the system does not know leaves the listener's.
func TestAMarineReportReadsTidesInTheLocationsZone(t *testing.T) {
	m := &snapshot.Marine{Buoy: "46224"}
	if r := marineReportOf(m, "America/New_York", 40.6, -73.9); !r.Known || r.TZ == nil || r.TZ.String() != "America/New_York" {
		t.Errorf("the report reads its tides in %v; want America/New_York", r.TZ)
	}
	if r := marineReportOf(m, "Not/AZone", 40.6, -73.9); !r.Known || r.TZ != nil {
		t.Errorf("an unknown zone set the tides' zone to %v; want none, the listener's", r.TZ)
	}
	if r := marineReportOf(nil, "America/New_York", 40.6, -73.9); r.Known {
		t.Error("an inland location composed a marine report")
	}
}

// TestASeismicReportIsUnknownUntilTheFeedAnswers is 0.19.5 Q2.4 (RN-5): a cold
// state is skipped, never read from an empty list in the present tense.
func TestASeismicReportIsUnknownUntilTheFeedAnswers(t *testing.T) {
	if r := seismicReportOf(&snapshot.SeismicState{}, 33, -117); r.Known {
		t.Error("a seismic state the feed has not answered is read as known")
	}
	if r := seismicReportOf(nil, 33, -117); r.Known {
		t.Error("no seismic state is read as known")
	}
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	if r := seismicReportOf(&snapshot.SeismicState{AsOf: at}, 33, -117); !r.Known || !r.State.AsOf.Equal(at) {
		t.Error("an answered seismic state is not read")
	}
}
