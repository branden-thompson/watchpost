package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAFinalTagIsRefusedOnAReleaseCandidate is RH-9 (D-3): a final tag with
// go.mod requiring a paired library's release candidate fails, naming it,
// and no other module's version; a pre-release tag, or a final tag on final
// libraries, passes.
func TestAFinalTagIsRefusedOnAReleaseCandidate(t *testing.T) {
	dir := t.TempDir()
	write := func(body string) string {
		p := filepath.Join(dir, "go.mod")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	rc := write("module example.invalid/w\n\ngo 1.26.10\n\nrequire (\n\tgithub.com/branden-thompson/go-ionomaps v0.1.0-rc.6\n\tgithub.com/branden-thompson/go-tuimaps v0.3.0\n\tgithub.com/ebitengine/oto/v3 v3.5.0-alpha.11 // indirect\n)\n")
	var out, errOut bytes.Buffer
	if code := run([]string{"v0.19.0", rc}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "go-ionomaps v0.1.0-rc.6") || strings.Contains(errOut.String(), "tuimaps") || strings.Contains(errOut.String(), "oto") {
		t.Errorf("a final tag on a release candidate: exit %d, %q", code, errOut.String())
	}
	if code := run([]string{"v0.19.0-rc.1", rc}, &out, &errOut); code != 0 {
		t.Errorf("a pre-release tag on a release candidate: exit %d", code)
	}
	final := write("module example.invalid/w\n\ngo 1.26.10\n\nrequire github.com/branden-thompson/go-ionomaps v0.1.0\n")
	if code := run([]string{"v0.19.0", final}, &out, &errOut); code != 0 {
		t.Errorf("a final tag on final libraries: exit %d", code)
	}
	if code := run(nil, &out, &errOut); code != 2 {
		t.Errorf("no tag: exit %d", code)
	}
}
