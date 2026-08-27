package tty

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// Spec: mock M-V1 + D-15 (layered keymap; only '?' locked) + D-19 (a/A/ctrl+a
// defaults; f/c live unit toggle) + R-12a. The dashboard reads ONLY the
// Snapshot (import lint). This is the B3 skeleton for the first D-21 UAT.

func f64(v float64) *float64 { return &v }

func snap() *snapshot.Snapshot {
	obs := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	return &snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion, GeneratedAt: obs,
		Locations: []snapshot.Location{{
			Label: "Oceanside, CA", Zip: "92057",
			Harmonized: snapshot.Conditions{Temp: f64(22.8), Condition: "clear",
				Source: snapshot.SourceInfo{Provider: "nws"}},
			Daily: []snapshot.Daily{
				{Date: "2026-08-24", TempMax: f64(23.9), TempMin: f64(17.2), Condition: "clear"},
				{Date: "2026-08-25", TempMax: f64(25.0), TempMin: f64(18.0), Condition: "rain"},
			},
			Alerts: []snapshot.Alert{{Event: "Extreme Heat Watch", Severity: "severe", Headline: "until Friday"}},
		}},
		Providers: []snapshot.ProviderStatus{{ID: "nws", Status: snapshot.ProviderOK, FetchedAt: obs}}, // answered: counts ✔ (REVIEW C5)
	}
}

func dash(t *testing.T) tea.Model {
	t.Helper()
	m, err := NewDashboard(Config{Version: "0.1.0-test"})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44}) // 10-row recent window (chrome 32 + inset 2, UAT 46)
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	return model
}

func TestUnitToggleIsLive(t *testing.T) {
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if v := m.View().Content; !strings.Contains(v, "23ºC") {
		t.Fatalf("c must live-swap to Celsius:\n%s", v)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if v := m.View().Content; !strings.Contains(v, "73ºF") {
		t.Fatal("f must swap back to Fahrenheit")
	}
}

func TestQuitAndHelpBindings(t *testing.T) {
	m := dash(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q must quit (default binding, D-15-swappable)")
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if v := m2.View().Content; !strings.Contains(v, "Watchpost Help") {
		t.Fatalf("? must open the help modal:\n%s", v)
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v := m3.View().Content; strings.Contains(v, "Watchpost Help") {
		t.Fatal("esc must close the help modal")
	}
}

func TestKeymapConflictRejectedAtBuild(t *testing.T) {
	// D-15: user overrides merge with validation; a config claiming '?' for
	// anything but help must fail construction, not silently win.
	_, err := NewDashboard(Config{Version: "t", KeyOverrides: term.KeyMap{
		"search": {Keys: []string{"?"}},
	}})
	if err == nil {
		t.Fatal("'?' override must be rejected (R-3)")
	}
}

// fakeHooks wires deterministic Resolve/Commit for flow tests (UAT 26).
type fakeHooks struct {
	resolved  snapshot.LocationRef
	committed [][2][]snapshot.LocationRef
}

func dashWithHooks(t *testing.T, h *fakeHooks) tea.Model {
	t.Helper()
	m, err := NewDashboard(Config{Version: "t",
		Resolve: func(q string) (snapshot.LocationRef, error) { return h.resolved, nil },
		Commit: func(w, r []snapshot.LocationRef) error {
			h.committed = append(h.committed, [2][]snapshot.LocationRef{w, r})
			return nil
		}})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	return model
}

// runCmd executes a command the way the program loop would: batches are
// flattened (since Q3 the shimmer tick may ride along with a hook's
// command) and tick messages are dropped, so a drain never loops on the
// animation. Running a tick costs its 300 ms once.
func runCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, runCmd(c)...)
		}
		return out
	}
	if _, isTick := msg.(tickMsg); msg == nil || isTick {
		return nil
	}
	return []tea.Msg{msg}
}

// drain runs cmd and feeds every message it yields back into the model
// until the loop is quiet.
func drain(t *testing.T, m tea.Model, cmd tea.Cmd) tea.Model {
	t.Helper()
	queue := runCmd(cmd)
	for i := 0; len(queue) > 0 && i < 64; i++ { // bounded (P10-02): a hook that answers itself forever is a test bug
		msg := queue[0]
		queue = queue[1:]
		var next tea.Cmd
		m, next = m.Update(msg)
		queue = append(queue, runCmd(next)...)
	}
	return m
}

// stripANSITest removes SGR sequences for column math in tests.
func stripANSITest(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}

type fakeRadio struct {
	mu     sync.Mutex
	tuned  []string
	stops  int
	vol    int
	repeat RepeatMode
	queue  []snapshot.LocationRef
	mode   RadioMode
}

func (f *fakeRadio) Tune(ref snapshot.LocationRef) {
	f.mu.Lock()
	f.tuned = append(f.tuned, ref.Label)
	f.mu.Unlock()
}

func (f *fakeRadio) Stop() { f.mu.Lock(); f.stops++; f.mu.Unlock() }

func (f *fakeRadio) SetVolume(pct int) { f.mu.Lock(); f.vol = pct; f.mu.Unlock() }

func (f *fakeRadio) SetRepeat(mode RepeatMode, queue []snapshot.LocationRef) {
	f.mu.Lock()
	f.repeat, f.queue = mode, queue
	f.mu.Unlock()
}

func (f *fakeRadio) SetMode(mode RadioMode) { f.mu.Lock(); f.mode = mode; f.mu.Unlock() }

func TestFailSoftUXRedTeam09(t *testing.T) {
	// Red-team 0.9.0 F1/F4/F10: a failed radio shows its reason where the
	// station was; adding a location already on the watchlist is refused
	// with a reason; a commit that fails to save reopens the modal with the
	// error instead of vanishing.
	fr := &fakeRadio{}
	commits := 0
	commitErr := fmt.Errorf("cannot write config.toml: permission denied — check the file's permissions")
	m, _ := NewDashboard(Config{Version: "t", Radio: fr, Commit: func([]snapshot.LocationRef, []snapshot.LocationRef) error { commits++; return nil }})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	// F1: the reason on the station line, in both player sizes.
	model, _ = model.Update(RadioStatusMsg{State: "failed", Detail: "no voice for linux/arm: install Piper", Volume: 55})
	d := model.(Dashboard)
	for _, compact := range []bool{false, true} {
		if v := stripANSITest(strings.Join(d.radioLines(d.opts(), compact), "\n")); !strings.Contains(v, "✘ no voice for linux/arm: install Piper") {
			t.Fatalf("failed state shows the reason (compact=%v):\n%s", compact, v)
		}
	}
	// F4: a duplicate add is refused before it reaches the commit.
	model, _ = model.Update(resolvedMsg{mode: "add", ref: refsOf(snap())[0]})
	if d := model.(Dashboard); !strings.Contains(d.addErr, "already on the watchlist") || commits != 0 {
		t.Fatalf("duplicate add refused with a reason: %q, commits %d", d.addErr, commits)
	}
	// F10: a failed save comes back into view.
	model, _ = model.Update(committedMsg{err: commitErr, what: "remove"})
	if d := model.(Dashboard); d.modal != modalAdd || !strings.Contains(d.addErr, "remove failed: cannot write") || d.addMode != "add" {
		t.Fatalf("a failed commit reopens the modal with the reason: showAdd=%v err=%q", d.modal == modalAdd, d.addErr)
	}
}

// setupHarness wires the Setup window's hooks to recorders (UAT 100).
type setupHarness struct {
	def    *snapshot.LocationRef
	key    string
	watch  []snapshot.LocationRef
	setups int
}

func (h *setupHarness) config() Config {
	return Config{
		Version: "0.1.0-test",
		Suggest: func(q string, limit int) []snapshot.LocationRef {
			if strings.HasPrefix(strings.ToLower(q), "oce") {
				return []snapshot.LocationRef{{Label: "Oceanside, CA", Tag: "OCEAN", Zip: "92057", Lat: 33.24, Lon: -117.29}}
			}
			return nil
		},
		Setup: func(def snapshot.LocationRef, key string) error {
			h.def, h.key, h.setups = &def, key, h.setups+1
			return nil
		},
		Commit: func(watch, recent []snapshot.LocationRef) error { h.watch = watch; return nil },
	}
}

func typeText(m tea.Model, text string) tea.Model {
	for _, r := range text {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}
