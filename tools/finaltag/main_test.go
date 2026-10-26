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
	// R2H-3: go.mod's layout cannot hide a requirement, and a replace of a
	// paired library is refused under a final tag, whatever it points at.
	for name, body := range map[string]string{
		"replaced by a candidate":  "module example.invalid/w\n\ngo 1.26.10\n\nrequire github.com/branden-thompson/go-ionomaps v0.1.0\n\nreplace github.com/branden-thompson/go-ionomaps => github.com/branden-thompson/go-ionomaps v0.1.0-rc.6\n",
		"replaced by a local tree": "module example.invalid/w\n\ngo 1.26.10\n\nrequire github.com/branden-thompson/go-tuimaps v0.3.0\n\nreplace github.com/branden-thompson/go-tuimaps => ../go-tuimaps\n",
		"CRLF line endings":        "module example.invalid/w\r\n\r\ngo 1.26.10\r\n\r\nrequire github.com/branden-thompson/go-ionomaps v0.1.0-rc.6\r\n",
		"a trailing space":         "module example.invalid/w\n\ngo 1.26.10\n\nrequire github.com/branden-thompson/go-ionomaps v0.1.0-rc.6 \n",
	} {
		errOut.Reset()
		if code := run([]string{"v0.19.0", write(body)}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "github.com/branden-thompson/go-") {
			t.Errorf("%s: exit %d, %q; want refused, naming the library", name, code, errOut.String())
		}
	}
}

// TestTheReleaseRunsFinaltag is R2H-3: the release workflow runs finaltag on
// its tag before anything is built, so deleting the step fails here.
func TestTheReleaseRunsFinaltag(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	wf := string(b)
	step, verify := strings.Index(wf, `run: go run ./tools/finaltag "$GITHUB_REF_NAME"`), strings.Index(wf, "run: make verify")
	if step < 0 || verify < 0 || step > verify {
		t.Errorf("release.yml does not run finaltag on its tag before make verify (step at %d, verify at %d)", step, verify)
	}
}
