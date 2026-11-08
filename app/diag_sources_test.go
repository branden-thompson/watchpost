package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/fire"
	"github.com/branden-thompson/watchpost/domains/fire/firms"
	"github.com/branden-thompson/watchpost/domains/fire/hms"
	"github.com/branden-thompson/watchpost/domains/fire/wfigs"
	"github.com/branden-thompson/watchpost/domains/seismic"
	"github.com/branden-thompson/watchpost/domains/seismic/usgs"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestDiagnosticsFindsEachFireAndQuakeSource is 0.19.5 Q2.13 (RN's minor list):
// the Diagnostics window reads each fire and quake provider by its type, and a
// provider the switch dropped would blank its line there.
func TestDiagnosticsFindsEachFireAndQuakeSource(t *testing.T) {
	lp := &livePipelines{providerFamilies: providerFamilies{
		fire:    []snapshot.Provider{hms.New(nil, "", fire.Rules{}), wfigs.New(nil, "", fire.Rules{}), firms.New(nil, "", "", fire.Rules{})},
		seismic: []snapshot.Provider{usgs.New(nil, "", seismic.Rules{})},
	}}
	src := lp.sources()
	for name, found := range map[string]bool{"HMS": src.hms != nil, "WFIGS": src.wfigs != nil, "FIRMS": src.firms != nil, "USGS": src.usgs != nil} {
		if !found {
			t.Errorf("Diagnostics does not find the %s provider", name)
		}
	}
}

// TestAPreviewIsRefusedWhileTheStationHoldsTheAir is 0.19.5 Q2.13 (RN's minor
// list, D-91): an audition mixes into the output, which on the console is the
// station's, so it is refused and the chooser says why.
func TestAPreviewIsRefusedWhileTheStationHoldsTheAir(t *testing.T) {
	var said []string
	d := &radioDeck{air: func() bool { return false }, note: func(s string) { said = append(said, s) }}
	d.PreviewVoice("Samantha")
	if len(said) != 1 || said[0] != "preview is unavailable while the station holds the air" {
		t.Errorf("a preview with the station on the air says %q", said)
	}
}
