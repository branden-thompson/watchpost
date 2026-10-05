package tty

// map_loops_test.go — PF-3: the Work loops in flight are bounded, and QA-11:
// a panic in a library call or in one of the map's commands marks the map
// failed rather than ending the station.

import (
	"context"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// pendingMap is the map open with tiles waiting: a zoom in leaves the view
// sharpening.
func pendingMap(t *testing.T) Dashboard {
	t.Helper()
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	d = settleMap(t, d)
	for range 4 {
		m, _ := d.Update(mapKeyMsg(t, "+"))
		d = m.(Dashboard)
		if d.mapWorkCmd() != nil {
			return d
		}
	}
	t.Fatal("no zoom left work pending: this measures nothing")
	return d
}

// TestTheWorkLoopsInFlightAreBounded is PF-3: every pan key, answer and
// menu key asks for Work, and each Work asks again while work waits - so a
// held key would run a loop a press. At most mapWorkLoops run; a loop past
// the bound is not started, and the ones running pick up the work.
func TestTheWorkLoopsInFlightAreBounded(t *testing.T) {
	d := pendingMap(t)
	calls := &[]string{}
	d.mapPane.calls = calls
	var cmds []tea.Cmd
	for range 3 * mapWorkLoops { // a held key's presses, each asking for Work before any has run
		if cmd := d.mapWorkCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	gate := make(chan struct{})
	var wg sync.WaitGroup
	msgs := make([]tea.Msg, len(cmds))
	for i, cmd := range cmds { // all at once, as Bubble Tea runs them
		wg.Add(1)
		go func() { defer wg.Done(); <-gate; msgs[i] = cmd() }()
	}
	d.mapPane.workers.loops.Add(int32(mapWorkLoops)) // as many loops as the bound already in Work
	close(gate)
	wg.Wait()
	works := 0
	for _, c := range *calls {
		if c == "Work" {
			works++
		}
	}
	if works != 0 {
		t.Errorf("%d loops ran Work past the bound of %d", works, mapWorkLoops)
	}
	if d.mapWorkCmd() != nil {
		t.Error("a Work command was made with the bound's loops running")
	}
	for _, msg := range msgs {
		if _, again := d.applyMapWorked(msg.(mapWorkedMsg)); again != nil {
			t.Error("a loop not started asked for Work again")
		}
	}
	d.mapPane.workers.loops.Add(-int32(mapWorkLoops)) // the running loops end, and the work is still theirs
	d = settleMap(t, d)
	if works := strings.Count(strings.Join(*calls, " "), "Work"); works == 0 {
		t.Error("the work waiting was never done")
	}
}

// TestAPanicInALibraryCallMarksTheMapFailed is QA-11: a call that panics is
// stopped; the map is let go and marked failed, the window says how to open
// it again, and the panic's words go to the diagnostics.
func TestAPanicInALibraryCallMarksTheMapFailed(t *testing.T) {
	var said []string
	d := mapDash(t, Config{MapProblem: func(p string) { said = append(said, p) }})
	d, _ = pressKey(d, "g")
	d = settleMap(t, d)
	d.mapPane.call("Render", func() { panic("a defect in the library") })
	m, _ := d.Update(mapTickMsg{})
	d = m.(Dashboard)
	assertMapFailed(t, d, said, "Render", "a defect in the library")
}

// TestAPanicInAMapCommandMarksTheMapFailed is QA-11 for the map's commands:
// Work - the one call made off the UI goroutine - and the commands that ask
// the app, which run under the same workers.
func TestAPanicInAMapCommandMarksTheMapFailed(t *testing.T) {
	for name, body := range map[string]func(p mapPane) func(context.Context) tea.Msg{
		"Work": func(p mapPane) func(context.Context) tea.Msg {
			return func(context.Context) tea.Msg {
				p.call("Work", func() { panic("work went wrong") })
				return mapWorkedMsg{}
			}
		},
		"command": func(mapPane) func(context.Context) tea.Msg {
			return func(context.Context) tea.Msg { panic("work went wrong") }
		},
	} {
		var said []string
		d := mapDash(t, Config{MapProblem: func(p string) { said = append(said, p) }})
		d, _ = pressKey(d, "g")
		d = settleMap(t, d)
		msg := runCmdElsewhere(d.mapPane.workers.cmd(context.Background(), nil, body(d.mapPane)))
		if msg == nil {
			t.Fatalf("%s: the command that panicked sent nothing, so no Update would see it", name)
		}
		m, _ := d.Update(msg)
		assertMapFailed(t, m.(Dashboard), said, name, "work went wrong")
	}
}

// assertMapFailed checks the map is let go and failed, the window says so,
// and the diagnostics hold the panic.
func assertMapFailed(t *testing.T, d Dashboard, said []string, where, words string) {
	t.Helper()
	if d.mapPane.m != nil || d.mapPane.failed != mapFailedText {
		t.Errorf("%s: after a panic the map is %v, failed %q; want it let go and failed", where, d.mapPane.m, d.mapPane.failed)
	}
	if body := strings.Join(d.mapBodyLines(), "\n"); !strings.Contains(body, mapFailedText) {
		t.Errorf("%s: the window does not say the map failed:\n%s", where, body)
	}
	if all := strings.Join(said, "\n"); !strings.Contains(all, where) || !strings.Contains(all, words) {
		t.Errorf("%s: the diagnostics were told %q", where, all)
	}
}
