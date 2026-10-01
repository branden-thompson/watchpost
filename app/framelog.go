package app

// framelog.go — the map's frame recorder (D-198): under
// WATCHPOST_DEBUG_MAPFRAMES=<file> every map frame drawn is appended to the
// file as a JSON line, for a live run to show what the screen showed (U2-46,
// U2-47). Without the switch there is no recorder and the dashboard gets no
// hook.

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/branden-thompson/watchpost/modes/tty"
)

// frameLog appends frames to its file.
type frameLog struct {
	mu  sync.Mutex
	enc *json.Encoder
}

// newFrameLog is a recorder writing to the file the switch names, or nil
// without it - or where the file cannot be opened (a debug switch: said on
// stderr, never the listener's).
func newFrameLog() *frameLog {
	path := os.Getenv("WATCHPOST_DEBUG_MAPFRAMES")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		_, _ = os.Stderr.WriteString("WATCHPOST_DEBUG_MAPFRAMES: " + err.Error() + "\n") // a debug switch: stderr is all there is
		return nil
	}
	return &frameLog{enc: json.NewEncoder(f)}
}

// hook is the dashboard's Config.MapFrame: nil without a recorder.
func (l *frameLog) hook() func(tty.MapFrame) {
	if l == nil {
		return nil
	}
	return l.note
}

// note appends one frame.
func (l *frameLog) note(f tty.MapFrame) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.enc.Encode(f) // a debug file: a failed write is a missing line
}
