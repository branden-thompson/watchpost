package report

// Quality pass Q2 (L3-F23): the plain report is the documented screen-reader
// surface (R-12d); this golden pins it byte for byte at the 80-column
// width, so a formatting drift is a visible diff. -update-golden re-captures
// after an intentional change.

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "re-capture testdata/plain-80.golden")

func TestPlainReportGoldenAt80Columns(t *testing.T) {
	got := RenderPlain(fixtureSnapshot(), 80)
	path := filepath.Join("testdata", "plain-80.golden")
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d bytes)", path, len(got))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no golden yet: run with -update-golden (%v)", err)
	}
	if got != string(want) {
		t.Fatalf("plain report drifted from the golden — if intended, re-run with -update-golden and say why in the build log:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
