package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestARowUnderItsFloorFails is D-264: an exempted density row whose package
// measures under the floor its reason states fails, naming it; one at or over
// passes; a row stating no floor and a live finding are not this check's; a
// finding whose figure cannot be read, or a report that is not JSON, fails
// closed.
func TestARowUnderItsFloorFails(t *testing.T) {
	row := func(file, density, reason string) string {
		return `{"file":"` + file + `","rule_id":"P10-05-INVARIANT-DENSITY","message":"P10-05: invariant density ` + density + ` < 2.0 across 9 changed function(s)","exempted":true,"exempt_reason":"` + reason + `"}`
	}
	for _, c := range []struct {
		name, findings string
		code           int
		says           string
	}{
		{"at its floor", row("a/a.go", "0.43", "FLOOR 0.43 over 339 changed functions"), 0, "1 density row(s) at or over"},
		{"over its floor", row("a/a.go", "0.50", "FLOOR 0.43 over 339"), 0, "1 density row(s) at or over"},
		{"under its floor", row("a/a.go", "0.42", "FLOOR 0.43 over 339"), 1, "a/a.go: 0.42, under its floor 0.43"},
		{"no floor stated", row("a/a.go", "0.10", "ratified by precedent"), 0, "0 density row(s)"},
		{"a live finding", `{"file":"a/a.go","rule_id":"P10-05-INVARIANT-DENSITY","message":"invariant density 0.10","exempted":false,"exempt_reason":"FLOOR 0.43"}`, 0, "0 density row(s)"},
		{"another rule", `{"file":"a/a.go","rule_id":"P10-01-CONTROL-FLOW","message":"recursion","exempted":true,"exempt_reason":"FLOOR 0.43"}`, 0, "0 density row(s)"},
		{"no figure measured", `{"file":"a/a.go","rule_id":"P10-05-INVARIANT-DENSITY","message":"P10-05: density unreadable","exempted":true,"exempt_reason":"FLOOR 0.43"}`, 2, "states no density"},
	} {
		p := filepath.Join(t.TempDir(), "p10.json")
		if err := os.WriteFile(p, []byte(`{"findings":[`+c.findings+`]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		code := run([]string{p}, &out, &errOut)
		if code != c.code || !strings.Contains(out.String()+errOut.String(), c.says) {
			t.Errorf("%s: exit %d, said %q; want %d and %q", c.name, code, out.String()+errOut.String(), c.code, c.says)
		}
	}
	p := filepath.Join(t.TempDir(), "p10.json")
	if err := os.WriteFile(p, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{p}, &out, &errOut); code != 2 {
		t.Errorf("a report that is not JSON: exit %d; want 2", code)
	}
	if code := run(nil, &out, &errOut); code != 2 {
		t.Errorf("no report named: exit %d; want 2", code)
	}
	out.Reset()
	if code := run([]string{"--self-test"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "(OK)") {
		t.Errorf("the self-test: exit %d, said %q; want it to pass in this tree", code, out.String())
	}
}
