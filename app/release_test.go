package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// A DEV BUILD IS NEVER BEHIND (0.14.0).
//
// The running version is `git describe`, so an untagged build reads
// "0.14.0-3-gabc1234-dirty" — a build BETWEEN releases. Telling somebody
// working on the next version that they are out of date would be both wrong and
// annoying, so only the numbers are compared and only when both sides have them.
func TestSemverLessIgnoresEverythingPastThePatch(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"0.14.0", "0.15.3", true},
		{"0.14.0", "0.14.1", true},
		{"0.14.0", "1.0.0", true},
		{"0.15.3", "0.14.0", false},
		{"0.14.0", "0.14.0", false},
		{"v0.14.0", "v0.15.0", true},                 // a tag with its v
		{"0.14.0-3-gabc1234-dirty", "0.14.0", false}, // a dev build is PAST 0.14.0
		{"0.14.0-3-gabc1234-dirty", "0.15.0", true},  // but still behind the next release
		{"0.0.0-dev", "0.15.0", true},                // the unstamped default
		{"abcdef1", "0.15.0", false},                 // a bare sha: no verdict
		{"0.14", "0.15.0", false},                    // not three parts: no verdict
	} {
		if got := semverLess(tc.a, tc.b); got != tc.want {
			t.Errorf("semverLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

// A FAILED CHECK SAYS NOTHING, and never claims freshness it has not confirmed.
// A dashboard that told a listener they were current because it failed to ask
// would be worse than one that never checked.
func TestAFailedOrUnansweredCheckClaimsNothing(t *testing.T) {
	w := newReleaseWatch("0.14.0", true)
	if _, latest, behind := w.Status(); latest != "" || behind {
		t.Errorf("before any answer: latest %q behind %v", latest, behind)
	}

	// A server that refuses: still nothing claimed, and no panic.
	down := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer down.Close()
	c, err := httpx.New(httpx.Config{UserAgent: "watchpost/test (t@example.com)", RatePerSec: 1000})
	if err != nil {
		t.Fatal(err)
	}
	w.checkAt(context.Background(), c, down.URL)
	if _, latest, behind := w.Status(); latest != "" || behind {
		t.Errorf("after a failure: latest %q behind %v", latest, behind)
	}

	// A draft or a pre-release is not an answer either — it is not out yet.
	for _, body := range []string{`{"tag_name":"v0.99.0","draft":true}`, `{"tag_name":"v0.99.0","prerelease":true}`, `{}`} {
		srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write([]byte(body))
		}))
		w.checkAt(context.Background(), c, srv.URL)
		srv.Close()
		if _, latest, _ := w.Status(); latest != "" {
			t.Errorf("%s must not count as a release, got %q", body, latest)
		}
	}

	// A real one does.
	ok := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write([]byte(`{"tag_name":"v0.15.3"}`))
	}))
	defer ok.Close()
	w.checkAt(context.Background(), c, ok.URL)
	running, latest, behind := w.Status()
	if running != "0.14.0" || latest != "0.15.3" || !behind {
		t.Errorf("a published release is the answer: %q %q %v", running, latest, behind)
	}
}

// THE UPDATE ROW SHOWS NUMBERS THIS APP PARSED, not text a third party wrote.
//
// The "vX available" line is the row a reader is most inclined to act on, so a
// published tag never reaches it verbatim: anything that is not three numbers
// is not a version, and a tag carrying escape sequences or unbounded text
// cannot be painted onto a terminal through this path.
func TestOnlyParsedVersionNumbersReachTheUpdateRow(t *testing.T) {
	for _, tc := range []struct {
		name, tag, want string
	}{
		{"an ordinary tag", "v0.15.0", "0.15.0"},
		{"no v prefix", "0.15.0", "0.15.0"},
		{"a pre-release suffix is dropped", "v0.15.0-rc1", "0.15.0"},
		{"a hyperlink in the tag", "v99.0.0-\x1b]8;;https://evil.example\aGET UPDATE\x1b]8;;\a", "99.0.0"},
		{"a clipboard write", "v9.9.9\x1b]52;c;cm0gLXJmIH4K\a", ""},
		{"not a version at all", "latest", ""},
		{"a hundred kilobytes of tag", "v1.0.0" + strings.Repeat("A", 100_000), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				body, err := json.Marshal(map[string]any{"tag_name": tc.tag})
				if err != nil {
					t.Error(err)
					return
				}
				_, _ = rw.Write(body)
			}))
			defer srv.Close()
			client, err := httpx.New(httpx.Config{UserAgent: "watchpost/test (t@example.com)", RatePerSec: 1000})
			if err != nil {
				t.Fatal(err)
			}
			watch := newReleaseWatch("0.14.0", true)
			watch.checkAt(context.Background(), client, srv.URL)
			_, latest, _ := watch.Status()
			if latest != tc.want {
				t.Errorf("latest = %q, want %q", latest, tc.want)
			}
			if strings.ContainsRune(latest, 0x1b) {
				t.Errorf("an escape sequence reached the update row: %q", latest)
			}
		})
	}
}
