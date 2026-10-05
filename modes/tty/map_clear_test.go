package tty

// map_clear_test.go — PF-4: Clear map data empties the live map's memory on
// the UI goroutine and leaves its disk to the app's command; IS-M5: what the
// purge could not do is in the answer; IS-M8: the listener is told a count
// and where the detail is, never an OS error with a home path.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// diskMapDash is the map open with a disk cache at dir holding one tile
// file, then closed into Settings on Clear map data.
func diskMapDash(t *testing.T, cfg Config, dir string) Dashboard {
	t.Helper()
	tile := filepath.Join(dir, "v1", "source", "3", "1-2.pbf")
	if err := os.MkdirAll(filepath.Dir(tile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tile, []byte("tile"), 0o600); err != nil {
		t.Fatal(err)
	}
	disk := func(m *tuimaps.Map) error { return m.CacheRoot(dir, 1<<20) }
	cfg.NewMap = func(size tuimaps.Size) (*tuimaps.Map, error) {
		m, err := embeddedMap(size)
		if err == nil {
			err = disk(m)
		}
		return m, err
	}
	if cfg.MapDisk == nil {
		cfg.MapDisk = disk
	}
	d := mapDash(t, cfg)
	d, _ = pressKey(d, "g")
	d = settleMap(t, d)
	d, _ = pressKey(d, "esc")
	return d.openSetupAt(rowMapClear)
}

// TestClearingLeavesTheDiskToTheAppsCommand is PF-4: the UI goroutine
// empties the live map's memory and unlinks no tile - the app's command
// purges the directory - and once the app has answered, the live map keeps
// its tiles on disk again.
func TestClearingLeavesTheDiskToTheAppsCommand(t *testing.T) {
	dir := t.TempDir()
	attached := 0
	d := diskMapDash(t, Config{ClearMapData: func() MapCleared { return MapCleared{Files: 1} }, MapDisk: func(m *tuimaps.Map) error {
		attached++
		return m.CacheRoot(dir, 1<<20)
	}}, dir)
	m, cmd, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = m.(Dashboard)
	if _, err := os.Stat(filepath.Join(dir, "v1", "source", "3", "1-2.pbf")); err != nil {
		t.Errorf("the UI goroutine unlinked the disk cache's tiles: %v", err)
	}
	if use := d.mapPane.m.CacheUse(); use.Disk.Limit != 0 {
		t.Error("the live map still holds the disk cache the app is emptying")
	}
	m, _ = d.Update(cmd())
	d = m.(Dashboard)
	if attached != 1 || d.mapPane.m.CacheUse().Disk.Limit == 0 {
		t.Errorf("after the app's answer the disk cache was named %d times; want the live map keeping tiles on disk again", attached)
	}
}

// TestClearingSaysACountAndWhereTheDetailIs is IS-M8 with IS-M5: what could
// not be cleared is a count and the way to the diagnostics, which hold each
// error; no OS error, and no home path, reaches the listener.
func TestClearingSaysACountAndWhereTheDetailIs(t *testing.T) {
	var said []string
	failed := errors.Join(errors.New("remove /Users/someone/Library/Caches/watchpost/map/v1/a.pbf: permission denied"),
		errors.New("remove /Users/someone/Library/Caches/watchpost/map-http/b: permission denied"))
	d := diskMapDash(t, Config{MapProblem: func(p string) { said = append(said, p) },
		ClearMapData: func() MapCleared { return MapCleared{Files: 7, Zones: 3, Err: failed} }}, t.TempDir())
	m, cmd, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = m.(Dashboard)
	d.mapPane.clearErr = errors.New("the map is closed") // the live map's own purge refused (IS-M5)
	m, _ = d.Update(cmd())
	notes := strings.Join(footerText(m.(Dashboard)), "\n")
	if strings.Contains(notes, "/Users/") || strings.Contains(notes, "permission denied") {
		t.Errorf("the listener is shown the OS's errors:\n%s", notes)
	}
	if !strings.Contains(notes, "3 could not be") || !strings.Contains(notes, "Diagnostics") {
		t.Errorf("the notice does not say how many failed and where the detail is:\n%s", notes)
	}
	if all := strings.Join(said, "\n"); !strings.Contains(all, "a.pbf: permission denied") || !strings.Contains(all, "map-http/b") || !strings.Contains(all, "the map is closed") {
		t.Errorf("the diagnostics do not hold each error: %q", all)
	}
}
