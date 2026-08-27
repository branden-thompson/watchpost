package invariant

import (
	"strings"
	"testing"
)

// Quality pass Q2 (L3-F27): the P10 keystone had no test of its own. Four
// cases pin the contract every caller relies on — a true condition is
// silent, a false one names the violation, an unnamed violation is itself
// an error, and the error never panics.
func TestCheckContract(t *testing.T) {
	cases := []struct {
		name string
		cond bool
		msg  string
		want string // "" = nil error
	}{
		{"holds", true, "unused", ""},
		{"violated, named", false, "widget count must be positive", "invariant violated: widget count must be positive"},
		{"violated, unnamed", false, "", "invariant violated: unnamed violation (Check called with empty msg)"},
		{"holds with empty msg", true, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Check(tc.cond, tc.msg)
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("a holding invariant returns nil, got %v", err)
			case tc.want != "" && (err == nil || err.Error() != tc.want):
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
	if err := Check(false, "x"); !strings.HasPrefix(err.Error(), "invariant violated: ") {
		t.Fatal("every violation carries the same prefix so logs and [S] can find them")
	}
}
