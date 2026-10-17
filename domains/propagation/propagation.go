// Package propagation is the Propagation mode's data source (0.19.0 W4.1):
// the one go-ionomaps object the process holds (FR-4.8), the fetcher it is
// given over the propagation client (FR-4.4, FR-4.5, FR-4.7), and its
// updates, one at a time - go-ionomaps runs one `Update` at a time itself, so
// an update asked while another runs waits for it and is answered from what
// it fetched (A-30).
// Nothing here fetches until an update is asked, and the window asks only
// once its mode is open and its acknowledgement shown (FR-4.2, D-81).
package propagation

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Getter is the propagation client: one ask, its reply as it came.
type Getter interface {
	Get(ctx context.Context, rawURL, etag, lastModified string) (httpx.Answer, error)
}

// Options are the service's settings: a clock for tests, the update's own
// deadline, and where each rate header's first sighting is noted.
type Options struct {
	Clock     func() time.Time
	Deadline  time.Duration // 0 is the architecture's 60 s
	Note      func(string)  // the diagnostics; nil notes nothing
	StateFile string        // where GIRO's throttle state is kept across restarts (D-213); "" keeps none
}

// updateDeadline is an update's own deadline: a GIRO burst of 39 took 36 s
// in PLAN (architecture, "One owner runs the update").
const updateDeadline = 60 * time.Second

// Result is one update's snapshot, or why there is none.
type Result struct {
	Snapshot ionomaps.Snapshot
	Err      error
}

// Service holds the one library object.
type Service struct {
	lib      *ionomaps.Library
	fetch    *fetcher
	deadline time.Duration
	state    string       // the throttle state's file, or "" (D-213)
	note     func(string) // the diagnostics

	stateMu sync.Mutex // one write of the state file at a time: two asks' updates end one after the other
}

// The service's refusals.
var (
	errNoClient   = errors.New("propagation: no client to fetch with")
	errNoDeadline = errors.New("propagation: an update's deadline cannot be negative")
	errNoService  = errors.New("propagation: the service was not made with New")
	errNoStatus   = errors.New("propagation: the reply has no HTTP status")
)

// newLibrary is go-ionomaps' constructor, as a value, so a call to it is not
// read as New's own (P10-01's bare-name match).
var newLibrary = ionomaps.New

// An HTTP status is from 100 to 599.
const (
	firstStatus = 100
	lastStatus  = 599
)

// New is the service over the propagation client. It fetches nothing.
func New(get Getter, o Options) (*Service, error) {
	if get == nil {
		return nil, errNoClient
	}
	if o.Deadline < 0 {
		return nil, errNoDeadline
	}
	f := &fetcher{get: get, hosts: map[string]bool{}, note: o.Note, seen: map[string]bool{}}
	lib, err := newLibrary(f, ionomaps.Options{Clock: o.Clock})
	if err != nil {
		return nil, err
	}
	for _, src := range ionomaps.Sources() { // the library's own hosts, before anything is fetched (D-113)
		for _, h := range src.Hosts {
			f.hosts[h] = true
		}
	}
	deadline := o.Deadline
	if deadline == 0 {
		deadline = updateDeadline
	}
	s := &Service{lib: lib, fetch: f, deadline: deadline, state: o.StateFile, note: o.Note}
	s.restoreState()
	return s, nil
}

// restoreState takes the throttle state the last run kept (D-213), so a
// restart within the hour of a burst asks GIRO nothing new; one that cannot
// be read or that go-ionomaps refuses is noted, and the library starts as
// new - there is nothing for a listener to do (D-124).
func (s *Service) restoreState() {
	if s.state == "" {
		return
	}
	b, err := os.ReadFile(s.state)
	if errors.Is(err, os.ErrNotExist) {
		return // a first run
	}
	if err == nil {
		err = s.lib.Restore(b)
	}
	if err == nil {
		return
	}
	if s.note == nil {
		return
	}
	s.note("Propagation: GIRO's throttle state was not taken, so it starts as new - " + err.Error())
}

// keepState writes the throttle state after an update, through a file
// beside it renamed into place, so a run killed mid-write leaves the last
// whole state; a failure is noted (D-124).
func (s *Service) keepState() {
	if s.state == "" {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	err := os.MkdirAll(filepath.Dir(s.state), 0o700)
	tmp := s.state + ".tmp"
	if err == nil {
		err = os.WriteFile(tmp, s.lib.State(), 0o600)
	}
	if err == nil {
		err = os.Rename(tmp, s.state)
	}
	if err == nil {
		return
	}
	if s.note == nil {
		return
	}
	s.note("Propagation: GIRO's throttle state was not kept - " + err.Error())
}

// Ask runs an update with the caller's options and sends its result: its
// own, never another caller's (BUILD red team C-2) - an update asked while
// one runs waits for it in go-ionomaps and is answered from what it fetched.
// ctx is the mode's: closing the mode ends the update it asked for; the
// update also ends at its own deadline. A key press never cancels it.
func (s *Service) Ask(ctx context.Context, opts ionomaps.UpdateOptions) <-chan Result {
	out := make(chan Result, 1)
	made := s != nil && s.lib != nil
	if !made {
		out <- Result{Err: errNoService}
		return out
	}
	go func() {
		uctx, cancel := context.WithTimeout(ctx, s.deadline)
		defer cancel()
		snap, err := s.lib.Update(uctx, opts)
		s.keepState()
		out <- Result{Snapshot: snap, Err: err}
	}()
	return out
}

// fetcher is go-ionomaps' Fetcher over the propagation client: https to the
// library's own hosts alone (FR-4.7, D-113), and of each reply only the
// status, the rate-related headers and the validators (FR-4.5, D-83).
type fetcher struct {
	get   Getter
	hosts map[string]bool
	note  func(string)

	mu   sync.Mutex
	seen map[string]bool // host + header, once noted
}

var errUnlisted = errors.New("propagation: the address is not https on a host go-ionomaps exports")

// rateHeaders are the rate-related headers read at run time (D-83).
var rateHeaders = []string{"Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset",
	"RateLimit-Limit", "RateLimit-Remaining", "RateLimit-Reset"}

// Fetch asks the client for an address on one of the library's hosts.
func (f *fetcher) Fetch(ctx context.Context, rawURL string, v ionomaps.Validators) (ionomaps.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ionomaps.Response{}, errUnlisted
	}
	host := u.Hostname()
	if u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return ionomaps.Response{}, errUnlisted // the listed host on its own port, with no credentials (I-6)
	}
	if !f.hosts[host] {
		return ionomaps.Response{}, errUnlisted
	}
	a, err := f.get.Get(ctx, rawURL, v.ETag, v.LastModified)
	if err != nil {
		return ionomaps.Response{}, err
	}
	answered := a.Status >= firstStatus && a.Status <= lastStatus
	if !answered {
		return ionomaps.Response{}, errNoStatus
	}
	return ionomaps.Response{Status: a.Status, Rate: f.rate(host, a.Header),
		Validators: ionomaps.Validators{ETag: a.Header.Get("ETag"), LastModified: a.Header.Get("Last-Modified")}, Body: a.Body}, nil
}

// rate is a reply's rate-related headers, each noted by host and name the
// first time it is seen (FR-4.5), never its value.
func (f *fetcher) rate(host string, h http.Header) ionomaps.RateHeaders {
	var r ionomaps.RateHeaders
	for _, name := range rateHeaders {
		v := h.Get(name)
		if v == "" {
			continue
		}
		f.mu.Lock()
		first := !f.seen[host+" "+name] && f.note != nil
		f.seen[host+" "+name] = true
		f.mu.Unlock()
		if first {
			f.note("Propagation: " + host + " sent " + name)
		}
		switch {
		case name == "Retry-After":
			r.RetryAfter = v
		case strings.HasSuffix(name, "-Limit"):
			r.Limit = v
		case strings.HasSuffix(name, "-Remaining"):
			r.Remaining = v
		case strings.HasSuffix(name, "-Reset"):
			r.Reset = v
		}
	}
	return r
}
