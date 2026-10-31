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
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Getter is the propagation client: one ask, its reply as it came.
type Getter interface {
	Get(ctx context.Context, rawURL, etag, lastModified string) (httpx.Answer, error)
}

// Options are the service's settings: a clock for tests, the update's own
// deadline, and where each rate header's first sighting is noted.
type Options struct {
	Clock     func() time.Time
	Deadline  time.Duration // 0 is the architecture's 60 s; under go-ionomaps' MinDeadline is refused, MinDeadline itself asks (D-298, D-301)
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

	stateMu sync.Mutex     // one write of the state file at a time: two asks' updates end one after the other
	running sync.WaitGroup // the updates in flight, which Settle waits for
}

// The service's refusals.
var (
	errNoClient   = errors.New("propagation: no client to fetch with")
	errNoDeadline = fmt.Errorf("propagation: an update's deadline is 0, for the architecture's 60 s, or at least %s, the least every rule of an update holds at (D-298, D-301)", ionomaps.MinDeadline)
	errNoService  = errors.New("propagation: the service was not made with New")
	errNoStatus   = errors.New("propagation: the reply has no HTTP status")
)

// An HTTP status is from 100 to 599.
const (
	firstStatus = 100
	lastStatus  = 599
)

// NewService is the service over the propagation client. It fetches nothing.
func NewService(get Getter, o Options) (*Service, error) {
	if get == nil {
		return nil, errNoClient
	}
	if o.Deadline < 0 || o.Deadline > 0 && o.Deadline < ionomaps.MinDeadline {
		return nil, errNoDeadline // too short for every rule of an update to hold (D-298, D-301)
	}
	f := &fetcher{get: get, hosts: map[string]bool{}, note: o.Note, seen: map[string]bool{}}
	lib, err := ionomaps.New(f, ionomaps.Options{Clock: o.Clock})
	if err != nil {
		return nil, err
	}
	for _, src := range ionomaps.Sources() { // the library's own hosts, before anything is fetched (D-113)
		for _, h := range src.Hosts {
			f.hosts[h] = true
		}
	}
	// A LIBRARY THAT NAMED NO HOST WOULD HAVE EVERY ASK REFUSED, in silence,
	// as an address not on its list: the service is refused at its making.
	listed := len(f.hosts)
	if err := invariant.Check(listed > 0, "propagation: go-ionomaps names no host to ask, so every address would be refused"); err != nil {
		return nil, err
	}
	deadline := o.Deadline
	if deadline == 0 {
		deadline = updateDeadline
	}
	s := &Service{lib: lib, fetch: f, deadline: deadline, state: o.StateFile, note: o.Note}
	s.restoreState()
	return s, nil
}

// restoreState takes the state the last run kept (D-213, D-249), so a
// restart within the hour of a burst asks GIRO nothing new and the kept
// solar flux builds a field; a part that cannot be read or that go-ionomaps
// refuses is noted, naming the part, and starts as new - there is nothing
// for a listener to do (D-124).
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
	s.note("Propagation: " + restoreWords(err) + " - " + fileOnly(err))
}

// restoreWords say which part of the kept state was not taken: go-ionomaps
// takes GIRO's throttle state and the solar flux each on its own, and names
// the part it refused (R3L-6).
func restoreWords(err error) string {
	giro, mean := errors.Is(err, ionomaps.ErrBadGIROState), errors.Is(err, ionomaps.ErrBadMean)
	if giro && mean {
		return "neither GIRO's throttle state nor the kept solar flux was taken, so both start as new"
	}
	if giro {
		return "GIRO's throttle state was not taken, so GIRO starts as new; the kept solar flux was taken"
	}
	if mean {
		return "the kept solar flux was not taken; GIRO's throttle state was"
	}
	return "the kept state was not taken, so GIRO's throttle and the solar flux start as new"
}

// fileOnly is an error with any file's path cut to its name: the state's
// path holds the listener's home, and diagnostics reach the dumps (RI-7).
func fileOnly(err error) string {
	if err == nil {
		return "" // nothing failed: nothing to say
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Op + " " + filepath.Base(pe.Path) + ": " + pe.Err.Error()
	}
	var le *os.LinkError
	if errors.As(err, &le) {
		return le.Op + " " + filepath.Base(le.Old) + " to " + filepath.Base(le.New) + ": " + le.Err.Error()
	}
	return err.Error()
}

// keepState writes the throttle state after an update, through a file of
// its own beside it renamed into place - so a run killed mid-write leaves
// the last whole state, and two runs never write through one file; a
// failure is noted (D-124).
func (s *Service) keepState() {
	if s.state == "" {
		return
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	err := os.MkdirAll(filepath.Dir(s.state), 0o700)
	var tmp *os.File
	if err == nil {
		tmp, err = os.CreateTemp(filepath.Dir(s.state), filepath.Base(s.state)+".*.tmp")
	}
	if err == nil {
		_, err = tmp.Write(s.lib.State())
		if cerr := tmp.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			err = os.Rename(tmp.Name(), s.state)
		}
		if err != nil {
			_ = os.Remove(tmp.Name()) // a failed write leaves no file of its own behind
		}
	}
	if err == nil {
		return
	}
	if s.note == nil {
		return
	}
	s.note("Propagation: GIRO's throttle state was not kept - " + fileOnly(err))
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
	s.running.Add(1)
	go func() {
		defer s.running.Done()
		uctx, cancel := context.WithTimeout(ctx, s.deadline)
		defer cancel()
		snap, err := s.lib.Update(uctx, opts)
		s.keepState()
		out <- Result{Snapshot: snap, Err: err}
	}()
	return out
}

// Settle waits up to a bound for the updates in flight to end and keep the
// throttle state (D-213), so a listener who quits during GIRO's burst - the
// app's context cancelled first, the update's ask stopping at once - leaves
// the hour it spent recorded; false when the bound passed first.
func (s *Service) Settle(bound time.Duration) bool {
	if s == nil {
		return true
	}
	done := make(chan struct{})
	go func() {
		s.running.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(bound):
		return false
	}
}

// fetcher is go-ionomaps' Fetcher over the propagation client: https to the
// library's own hosts alone (FR-4.7, D-113), and of each reply only the
// status, the rate-related headers and the validators (FR-4.5, D-83).
type fetcher struct {
	get   Getter
	hosts map[string]bool
	note  func(string)

	mu   sync.Mutex
	seen map[string]bool // host + header, or host + status, once noted
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
	f.noteStatus(host, a.Status)
	// A validator past maxValidator is not kept (RI-4): the next ask asks
	// whole, never echoing the service's bytes back.
	kept := ionomaps.Validators{ETag: a.Header.Get("ETag"), LastModified: a.Header.Get("Last-Modified")}
	if len(kept.ETag) > maxValidator {
		kept.ETag = ""
	}
	if len(kept.LastModified) > maxValidator {
		kept.LastModified = ""
	}
	return ionomaps.Response{Status: a.Status, Rate: f.rate(host, a.Header), Validators: kept, Body: a.Body}, nil
}

// namedStatus are the single statuses D-247, D-260 and D-272 name: an
// answer, an unchanged file's 304, a denial (401, 403), a station or file
// gone (404, 410) and a refusal (429); a failure on the server's side (5xx)
// is named as a class. Any other is still a station's or a file's own
// failure, and is noted once (D-272).
var namedStatus = map[int]bool{200: true, 304: true, 401: true, 403: true, 404: true, 410: true, 429: true}

// noteStatus notes a status D-247, D-260 and D-272 do not name, by host and
// status, the first time the host answers it, in words for either service.
func (f *fetcher) noteStatus(host string, status int) {
	if namedStatus[status] {
		return
	}
	if status >= 500 && status <= 599 {
		return // a server's failure, which D-247 and D-260 name
	}
	if f.note == nil {
		return // no diagnostics to note into
	}
	key := host + " status " + strconv.Itoa(status)
	f.mu.Lock()
	first := !f.seen[key]
	f.seen[key] = true
	f.mu.Unlock()
	if first {
		f.note("Propagation: " + host + " answered " + strconv.Itoa(status) + ", a status it is not known to send: that ask's station or file is taken as missed (D-272)")
	}
}

// maxValidator is the longest validator kept and sent back: an ETag or a
// date runs to tens of bytes, and a longer one would be echoed on every ask.
const maxValidator = 256

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
