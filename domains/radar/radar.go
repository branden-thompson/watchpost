// Package radar is the map's radar sources (0.18.0 W8, FR-5): the Iowa
// Environmental Mesonet's N0Q composite and NOAA/NCEP's MRMS reflectivity,
// each asked for its advertised times and for frames of a fixed radar box -
// never the listener's view (D-47).
//
// THE TRAPS ARE SHUT AT THE REQUEST (wave 1 W1-A; D-84). Both sources answer
// HTTP 200 with an empty picture for a time they do not hold, and IEM draws
// radar from 2011 when no time is sent. So a frame is only ever asked for at a
// time the source advertised, and always with that time: Frame refuses any
// other. An empty picture at an advertised time is a real frame with no echo.
package radar

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Source is one radar source (W8.3).
type Source interface {
	Name() string
	// Covers reports whether the source draws a map region (geo's names).
	Covers(region string) bool
	// Times is the source's advertised frame times for a region, oldest first.
	Times(ctx context.Context, region string) ([]time.Time, error)
	// Frame is one PNG of a box at an advertised time.
	Frame(ctx context.Context, region string, at time.Time, advertised []time.Time, b Box) ([]byte, error)
}

// Getter is the one call a source makes: the radar client's (NewClient).
type Getter interface {
	GetText(ctx context.Context, rawURL string, opts ...httpx.Option) ([]byte, error)
}

// ErrNotAdvertised is a frame asked for at a time its source did not list.
var ErrNotAdvertised = errors.New("radar: a frame was asked for at a time its source does not advertise")

// ErrNotCovered is a region the source does not draw.
var ErrNotCovered = errors.New("radar: the source does not cover this region")

// advertisedOrErr refuses a time the source did not list.
func advertisedOrErr(at time.Time, advertised []time.Time) error {
	if at.IsZero() || !slices.ContainsFunc(advertised, at.Equal) {
		return ErrNotAdvertised
	}
	return nil
}

// Window is the loop's length: two hours (W8.3b, FR-3.9's radar retention).
const Window = window

// frameBodyCap is the most a frame may be (FR-5.7): 1 MiB against measured
// frames of 7.7 to 30 KB.
const frameBodyCap = 1 << 20

// NewClient is the radar client (W8.5, W8.14, RK-11, D-55): memory only - no
// radar request is ever written to disk - a 1 MiB body cap enforced as it
// reads, https only, and no dial to a private address.
func NewClient(userAgent string) (*httpx.Client, error) {
	return httpx.New(ClientConfig(userAgent))
}

// ClientConfig is the radar client's configuration, named so a test holds it.
func ClientConfig(userAgent string) httpx.Config {
	return httpx.Config{UserAgent: userAgent, MaxRetries: 1, CacheDir: "",
		MaxBodyBytes: frameBodyCap, RefusePrivate: true, HTTPSOnly: true}
}
