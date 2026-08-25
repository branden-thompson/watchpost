package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestRestoreRecentKeepsTheSavedStackAboveTheSeeds(t *testing.T) {
	// UAT 96: the saved RECENT stack comes back on top in its saved order;
	// the seeds fill below; anything now a favourite drops out; duplicates
	// by zip collapse; the cap holds.
	ref := func(label, zip string) snapshot.LocationRef { return snapshot.LocationRef{Label: label, Zip: zip} }
	saved := []snapshot.LocationRef{ref("El Cajon, CA", "92020"), ref("Boise, ID", "83702"), ref("Oceanside, CA", "92057")}
	watch := []snapshot.LocationRef{ref("Oceanside, CA", "92057")}
	seeds := []snapshot.LocationRef{ref("New York, NY", "10001"), ref("Boise, ID", "83702"), ref("Chicago, IL", "60601"), ref("Houston, TX", "77002")}
	got := restoreRecent(saved, watch, seeds, 4)
	want := []string{"92020", "83702", "10001", "60601"}
	if len(got) != len(want) {
		t.Fatalf("got %d refs %v, want %v", len(got), got, want)
	}
	for i, zip := range want {
		if got[i].Zip != zip {
			t.Fatalf("position %d: got %s want %s (%v)", i, got[i].Zip, zip, got)
		}
	}
	if r := restoreRecent(nil, nil, seeds, 10); len(r) != 4 || r[0].Zip != "10001" {
		t.Fatalf("no history: the seeds alone, in order: %v", r)
	}
}

func TestCommitPersistsFavouritesAndTheRecentStack(t *testing.T) {
	// UAT 96: every commit writes both lists — favourites and the RECENT
	// stack, newest first — so a lookup survives quit and relaunch.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seed := config.Default()
	seed.Locations = []config.Location{{Label: "Oceanside, CA", Zip: "92057", Lat: 33.24, Lon: -117.29}}
	if err := config.Save(seed); err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{} // no pipelines: persistence alone
	watch := []snapshot.LocationRef{{Label: "Oceanside, CA", Zip: "92057", Lat: 33.24, Lon: -117.29, TZ: "America/Los_Angeles"}}
	recent := []snapshot.LocationRef{{Label: "El Cajon, CA", Zip: "92020", Lat: 32.79, Lon: -116.96}, {Label: "Boise, ID", Zip: "83702", Lat: 43.62, Lon: -116.2}}
	if err := lp.commit(watch, recent); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Locations) != 1 || cfg.Locations[0].Zip != "92057" || cfg.Locations[0].Tag == "" {
		t.Fatalf("favourites persisted with a derived tag: %+v", cfg.Locations)
	}
	if len(cfg.Recent) != 2 || cfg.Recent[0].Zip != "92020" || cfg.Recent[1].Zip != "83702" || cfg.Recent[0].Label != "El Cajon, CA" {
		t.Fatalf("the recent stack persisted newest first: %+v", cfg.Recent)
	}
	if got := refsFromConfig(cfg.Recent); got[0].Zip != "92020" || got[0].Lat != 32.79 {
		t.Fatalf("round trip through the one conversion pair: %+v", got)
	}
}

func TestLoadUserThemesRegistersJSONFiles(t *testing.T) {
	// UAT 53: <config>/themes/<name>.json registers a theme (token overrides).
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Ocean.json"), []byte(`{"tokens": {"temp.hi": "39"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "broken.json"), []byte(`not json`), 0o644)
	loadUserThemes(dir)
	defer render.SetTheme(render.DefaultThemeName)
	if !render.SetTheme("Ocean") || render.Tok(render.TempHi) != "39" {
		t.Fatalf("user theme must register and resolve: %q", render.Tok(render.TempHi))
	}
	if render.SetTheme("broken") {
		t.Fatal("unreadable theme files must be skipped")
	}
}
