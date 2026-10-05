package httpx

// brief.go — a failure as the diagnostics keep it (IS-M6): the host and the
// HTTP status, or the kind of failure, never the address asked. The path and
// query name stations, cities and zones, and the diagnostics leave the
// machine in a dump's counters.json.

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"syscall"
)

// ErrNotPublic is the dial check's refusal of an address that is not on the
// public internet; a dial error carries it, for errors.Is.
var ErrNotPublic = errors.New("not a public address")

// Brief is err said for the diagnostics: "<host> HTTP <status>" for a status
// failure, "<host> <kind>" for one with no status, "timed out" or
// "cancelled" for a context's end, else "failed (<type>)" naming the
// innermost error's Go type. "" for no error.
func Brief(err error) string {
	if err == nil {
		return ""
	}
	var se *StatusError
	if errors.As(err, &se) {
		return fmt.Sprintf("%s HTTP %d", se.Endpoint(), se.Status)
	}
	var re *ReachError
	if errors.As(err, &re) {
		return re.Endpoint() + " " + reachKind(re.Err)
	}
	if k := contextKind(err); k != "" {
		return k
	}
	inner := err
	for next := errors.Unwrap(inner); next != nil; next = errors.Unwrap(inner) { // a wrap chain is finite (P10-02)
		inner = next
	}
	return fmt.Sprintf("failed (%T)", inner)
}

// contextKind is a context's end said as such, or "".
func contextKind(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timed out"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	}
	return ""
}

// reachKind is why a host could not be reached, in a few words.
func reachKind(err error) string {
	var dns *net.DNSError
	var cert *tls.CertificateVerificationError
	var ne net.Error
	switch {
	case errors.Is(err, ErrNotPublic):
		return "refused: not a public address"
	case errors.As(err, &dns):
		return "name not resolved"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "connection refused"
	case errors.Is(err, syscall.ECONNRESET):
		return "connection reset"
	case errors.As(err, &cert):
		return "certificate not trusted"
	case contextKind(err) != "":
		return contextKind(err)
	case errors.As(err, &ne) && ne.Timeout():
		return "timed out"
	}
	return "unreachable"
}

// anyURL is an http or https address inside some text, up to a space or a
// quote.
var anyURL = regexp.MustCompile(`https?://[^\s"'<>]+`)

// ScrubURLs is s with every http or https address in it cut to its scheme and
// host: what text from outside this package - a library's error, a parser's -
// keeps of an address before it reaches the diagnostics.
func ScrubURLs(s string) string {
	return anyURL.ReplaceAllStringFunc(s, func(m string) string {
		u, err := url.Parse(m)
		if err != nil || u.Host == "" {
			return "<address>"
		}
		return u.Scheme + "://" + u.Host
	})
}
