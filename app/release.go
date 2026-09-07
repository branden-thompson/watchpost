package app

// release.go — "is there a newer Watchpost?" (0.14.0).
//
// One unauthenticated GET to the GitHub releases API, once an hour, through the
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

// releaseEvery is how often the check runs. An hour is far inside GitHub's
// unauthenticated budget (60 requests an hour per address) and far more often
// than releases happen; the point is that a long-running dashboard notices,
// not that it notices quickly.
const releaseEvery = time.Hour

// releaseWatch holds the last answer, for [S] to read.
type releaseWatch struct {
	mu      sync.Mutex
	running string // this binary, as it was stamped
	latest  string // the newest published tag, "" until one is known
	on      bool   // the listener asked for the check (config: update_check)
}

func newReleaseWatch(running string, on bool) *releaseWatch {
	return &releaseWatch{running: running, on: on}
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

// start polls until ctx ends. The first check runs at once so a listener who
// opens [S] in the first minute sees an answer rather than a blank.
func (w *releaseWatch) start(ctx context.Context, c *httpx.Client) {
	if !w.on {
		return // opt-in: no goroutine, no request, nothing to disclose
	}
	go func() {
		// CHECKS ONCE FIRST, then on the interval. The immediate pass is stated
		// here rather than passed to everyTick as a flag: a caller reading this
		// can see when the first request goes out.
		w.check(ctx, c)
		everyTick(ctx, releaseEvery, func(time.Time) { w.check(ctx, c) })
	}()
}

// check asks once. Every failure is silent by design — see the file header.
func (w *releaseWatch) check(ctx context.Context, c *httpx.Client) {
	w.checkAt(ctx, c, releaseAPI)
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
