package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// THE TAB SHOWN READS AS FOCUSED (UAT-2 U2-49): its pointer, name and brackets
// are the focus yellow a focused row's are, so the tab is found at a glance
// - and a held tab key can be watched stepping. Every other tab stays plain.
func TestTheShownTabIsInTheFocusColour(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	d := setupGolden(t, 133, 44, false, rowMapTempSource)
	o := d.opts()
	row := d.setupTabRow(o, 200)
	if want := render.ListLabel("[ "+o.Glyphs().Pointer+" Maps ]", true); !strings.Contains(row, want) {
		t.Errorf("the Maps tab is not in the focus colour:\n%q\nwant it to hold %q", row, want)
	}
	if narrow := d.setupTabRow(o, 70); !strings.Contains(narrow, render.ListLabel("["+o.Glyphs().Pointer+"Maps]", true)) {
		t.Errorf("the narrow row's Maps tab is not in the focus colour:\n%q", narrow)
	}
	if focus := render.ListLabel("[ Data ]", true); strings.Contains(row, focus) {
		t.Error("a tab not shown is in the focus colour")
	}
	if got := stripANSITest(row); !strings.Contains(got, "[ "+o.Glyphs().Pointer+" Maps ]") || !strings.Contains(got, "[ Data ]") {
		t.Errorf("the tab row's words changed: %q", got)
	}
}
