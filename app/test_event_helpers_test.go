package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/script"
)

// brokenLibrary is a script library whose breaking parts do not render — the
// worst case an edit to a user script can produce.
func brokenLibrary(t *testing.T) *script.Library {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "breaking"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"single.txt", "burst-line.txt"} {
		if err := os.WriteFile(filepath.Join(dir, "breaking", part), []byte("{{.Line"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return script.New(dir)
}
