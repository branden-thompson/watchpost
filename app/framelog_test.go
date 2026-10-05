package app

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty"
)

// THE FRAME RECORDER IS OFF UNLESS ASKED, AND APPENDS A LINE A FRAME (D-198).
func TestTheFrameRecorderIsOffUnlessAsked(t *testing.T) {
	t.Setenv("WATCHPOST_DEBUG_MAPFRAMES", "")
	if newFrameLog().hook() != nil {
		t.Fatal("with no switch, the dashboard was given a recorder")
	}
	path := filepath.Join(t.TempDir(), "frames.log")
	t.Setenv("WATCHPOST_DEBUG_MAPFRAMES", path)
	hook := newFrameLog().hook()
	hook(tty.MapFrame{LoopIndex: 3, RadarCells: 7, Places: []string{"Denver"}})
	hook(tty.MapFrame{LoopIndex: 4})
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var got []tty.MapFrame
	for sc := bufio.NewScanner(f); sc.Scan(); {
		var fr tty.MapFrame
		if err := json.Unmarshal(sc.Bytes(), &fr); err != nil {
			t.Fatal(err)
		}
		got = append(got, fr)
	}
	if len(got) != 2 || got[0].RadarCells != 7 || got[0].Places[0] != "Denver" || got[1].LoopIndex != 4 {
		t.Errorf("the file holds %+v; want the two frames, in order", got)
	}
}
