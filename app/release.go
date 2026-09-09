package app

// release.go — "is there a newer Watchpost?" (0.14.0).
//
// One unauthenticated GET to the GitHub releases API, once at startup, through the
// same client every provider uses — so it is paced, cached, retried and
// redacted like any other fetch, and it shows up in [S]'s own table as a host
// with traffic and no provider, which is exactly what it is.
//
// IT SENDS NOTHING. A GET carries the user agent and no more: no version, no
// identifier, no telemetry of any kind. The check is one-way — the app asks
// what the latest release is and compares locally.
//
// AND IT NEVER RAISES A WARNING. A failed update check is not a degraded
// provider: the app works exactly as well without it, so a listener who is
// offline, or behind a proxy that blocks GitHub, must not find an issue in the
// window that tells them their data is unhealthy. It simply reads "unknown".

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// releaseAPI is the latest-release endpoint for this repository.
const releaseAPI = "https://api.github.com/repos/branden-thompson/watchpost/releases/latest"

// releaseWatch holds the last answer, for [S] to read.
type releaseWatch struct {
	mu      sync.Mutex
	running string // this binary, as it was stamped
	latest  string // the newest published tag, "" until one is known
	on      bool   // the listener asked for the check (config: update_check)
	// api is the endpoint, overridable ONLY by a test. It exists because the
	// one-shot bound (FR-7.1) is not reachable otherwise: `start` is the thing
	// being bounded, and with a hardcoded URL a test can only exercise
	// `checkAt` and then claim something about `start` that it never ran. That
	// claim was written, and two planted defects walked straight past it.
	api string
}

func newReleaseWatch(running string, on bool) *releaseWatch {
	return &releaseWatch{running: running, on: on, api: releaseAPI}
}

// Enabled reports whether the check runs at all.
func (w *releaseWatch) Enabled() bool { return w != nil && w.on }

// Status is what [S] shows: the running version, the newest one when it is
// known, and whether this binary is behind it.
func (w *releaseWatch) Status() (running, latest string, behind bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running, w.latest, w.latest != "" && semverLess(w.running, w.latest)
}

// start asks ONCE, at startup, and is then done (FR-7.1, HUM LEAD 2026-09-08).
//
// It used to poll hourly, and that poller is what made this file look like a
// data feed: a goroutine, an interval, a cancellation path — the shape of a
// provider, on something that is not one. Deleting it is what makes the ruling
// visible in the code rather than only in the architecture note.
//
// IT ALSO RETIRES A P10 EXEMPTION rather than carrying one. The ledger row for
// this function was granted because an hourly ticker is "an unbounded event
// loop by nature… no meaningful iteration count to bound it by". One check has
// a bound of one.
//
// WHAT THIS GIVES UP, SAID PLAINLY: a dashboard left running for weeks will not
// notice a release published while it was up. That was the old comment's stated
// point. It is accepted because ACTING on the notice needs a restart anyway, so
// once-at-startup reports it at the moment the listener can do something about
// it. If that ever proves wrong, the fix is a re-check when [S] OPENS — not a
// timer — and it needs a don't-refetch-within guard, which is a slice of the
// state being deleted here. Recorded so the next person weighs it rather than
// rediscovering it.
func (w *releaseWatch) start(ctx context.Context, c *httpx.Client) {
	if !w.on {
		return // opt-in: no goroutine, no request, nothing to disclose
	}
	go w.check(ctx, c) // off the startup path; the answer lands when it lands
}

// check asks once. Every failure is silent by design — see the file header.
func (w *releaseWatch) check(ctx context.Context, c *httpx.Client) {
	w.checkAt(ctx, c, w.api)
}

// checkAt is check against a given URL — the seam the tests point at a stub,
// so the suite never reaches GitHub.
func (w *releaseWatch) checkAt(ctx context.Context, c *httpx.Client, url string) {
	if c == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	body, err := c.GetText(ctx, url)
	if err != nil {
		return
	}
	var rel struct {
		Tag   string `json:"tag_name"`
		Draft bool   `json:"draft"`
		Pre   bool   `json:"prerelease"`
	}
	if json.Unmarshal(body, &rel) != nil || rel.Draft || rel.Pre || rel.Tag == "" {
		return
	}
	// THE PARSED NUMBERS ARE STORED, never the tag as published. The tag is
	// text from a third party that this app paints into an "update available"
	// line — the one row a reader is most inclined to act on — so it is reduced
	// to the three numbers a version comparison actually uses. A tag carrying
	// escape sequences, a hyperlink, or a hundred kilobytes of anything cannot
	// reach a terminal through here.
	major, minor, patch, ok := releaseNumbers(rel.Tag)
	if !ok {
		return
	}
	w.mu.Lock()
	w.latest = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	w.mu.Unlock()
}

// releaseNumbers is a published tag reduced to the numbers a version is made
// of, and whether it was a version at all.
func releaseNumbers(tag string) (major, minor, patch int, ok bool) {
	parts, ok := semverParts(tag)
	if !ok {
		return 0, 0, 0, false
	}
	return parts[0], parts[1], parts[2], true
}

// semverLess reports whether a is an earlier release than b.
//
// ONLY THE NUMBERS ARE COMPARED, and only when both sides have three of them.
// The running version is `git describe`, so a build between releases reads
// "0.14.0-3-gabc1234-dirty"; everything past the patch — a pre-release, a commit
// count, a dirty marker — says this build is PAST that release, not before it,
// so it is ignored rather than treated as a version of its own. A build between
// 0.14.0 and 0.15.0 does read as behind once 0.15.0 publishes, which is true.
func semverLess(a, b string) bool {
	av, aok := semverParts(a)
	bv, bok := semverParts(b)
	if !aok || !bok {
		return false
	}
	for i := range av {
		if av[i] != bv[i] {
			return av[i] < bv[i]
		}
	}
	return false
}

// semverParts is the major/minor/patch of a version, and whether it had them.
// Anything after the patch — a pre-release, a commit count, a dirty marker — is
// ignored: it says this build is past that release, not before it.
func semverParts(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
