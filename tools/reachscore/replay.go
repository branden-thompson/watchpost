package main

// replay.go - the days' recorded NOAA and GIRO files, answered to
// go-ionomaps as the services would have answered at the replay's clock,
// and nothing dated after it.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// The addresses go-ionomaps asks (its `update.go` and `internal/giro`).
// The replay answers these alone: any other ask fails the run, so a
// library that has moved an address is caught, not answered with nothing.
const (
	glotecIndex = "https://services.swpc.noaa.gov/products/glotec/geojson_2d_urt/"
	solarFile   = "https://services.swpc.noaa.gov/text/daily-solar-indices.txt"
	drapTable   = "https://services.swpc.noaa.gov/text/drap_global_frequencies.txt"
	scalesFile  = "https://services.swpc.noaa.gov/products/noaa-scales.json"
	fastChar    = "https://lgdc.uml.edu/fastchar/getbest?"
)

// The statuses the replay answers with.
const (
	replayOK      = 200
	replayMissing = 404
)

// gridName is a GloTEC grid's file name; its stamp is the valid time.
var gridName = regexp.MustCompile(`^glotec_icao_(\d{8}T\d{6}Z)\.geojson$`)

// giroName is a recorded GIRO reply's file name, by its station's code.
var giroName = regexp.MustCompile(`^fc_([A-Z0-9]{5})\.txt$`)

// recorded is the days' files: the grids by name and valid time, the
// stations' replies by code, and the solar file; every path read, for the
// checksums.
type recorded struct {
	dir   string
	grids map[string]time.Time
	giro  map[string][]byte
	solar []byte
	files []string
}

// errNoInputs is a folder holding no grid or no reply, or no solar file.
var errNoInputs = errors.New("the inputs folder holds no GloTEC grid (glotec_icao_*.geojson) or no GIRO reply (fc_*.txt), or the solar file is empty")

// loadRecorded reads the folder's grids' names and replies, and the solar
// file.
func loadRecorded(dir, solar string) (*recorded, error) {
	if dir == "" || solar == "" {
		return nil, errUsage
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	in := &recorded{dir: dir, grids: map[string]time.Time{}, giro: map[string][]byte{}}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if m := gridName.FindStringSubmatch(e.Name()); m != nil {
			t, err := time.Parse("20060102T150405Z", m[1])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", e.Name(), err)
			}
			in.grids[e.Name()], in.files = t, append(in.files, path)
		}
		if m := giroName.FindStringSubmatch(e.Name()); m != nil {
			body, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			in.giro[m[1]], in.files = body, append(in.files, path)
		}
	}
	if in.solar, err = os.ReadFile(solar); err != nil {
		return nil, err
	}
	in.files = append(in.files, solar)
	if len(in.grids) == 0 || len(in.giro) == 0 || len(in.solar) == 0 {
		return nil, errNoInputs
	}
	slices.Sort(in.files)
	return in, nil
}

// replay is a go-ionomaps fetcher answering from the recorded files at its
// clock; grids false withholds GloTEC (the climatology's column).
type replay struct {
	in      *recorded
	now     time.Time
	grids   bool
	unknown []string
}

// clock is the replay's time, the library's clock.
func (r *replay) clock() time.Time { return r.now }

// errUnknownAsk is an address the replay holds no answer for.
var errUnknownAsk = errors.New("the replay holds no answer for this address")

// Fetch answers an ask as the service would have at the clock.
func (r *replay) Fetch(_ context.Context, u string, _ ionomaps.Validators) (ionomaps.Response, error) {
	if r.in == nil {
		return ionomaps.Response{}, errUnknownAsk
	}
	if u == "" {
		return ionomaps.Response{}, errUnknownAsk
	}
	switch {
	case u == glotecIndex:
		return r.index(), nil
	case strings.HasPrefix(u, glotecIndex):
		return r.grid(strings.TrimPrefix(u, glotecIndex))
	case u == solarFile:
		return answer(solarBefore(r.in.solar, r.now)), nil
	case u == drapTable, u == scalesFile:
		return ionomaps.Response{Status: replayMissing}, nil // none was kept for the days: no disturbance is said
	case strings.HasPrefix(u, fastChar):
		return r.station(u)
	}
	r.unknown = append(r.unknown, u)
	return ionomaps.Response{}, errUnknownAsk
}

// answer is a 200 with a body.
func answer(body []byte) ionomaps.Response {
	if body == nil {
		return ionomaps.Response{Status: replayMissing} // nothing held to serve
	}
	return ionomaps.Response{Status: replayOK, Body: body}
}

// index is GloTEC's directory index at the clock: the grids valid by then,
// as links, as NOAA's index names them.
func (r *replay) index() ionomaps.Response {
	if !r.grids {
		return ionomaps.Response{Status: replayMissing}
	}
	if r.in == nil {
		return ionomaps.Response{Status: replayMissing}
	}
	var b bytes.Buffer
	for _, name := range r.gridsBy() {
		fmt.Fprintf(&b, "<a href=%q>%s</a>\n", name, name)
	}
	return answer(b.Bytes())
}

// gridsBy are the grids' names valid at or before the clock, in order.
func (r *replay) gridsBy() []string {
	if r.in == nil {
		return nil
	}
	var out []string
	for name, valid := range r.in.grids {
		if !valid.After(r.now) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// grid is a grid's file; missing when withheld, unknown, or valid after
// the clock.
func (r *replay) grid(name string) (ionomaps.Response, error) {
	if r.in == nil {
		return ionomaps.Response{Status: replayMissing}, nil
	}
	if name == "" {
		return ionomaps.Response{Status: replayMissing}, nil // the index itself, asked without its slash
	}
	valid, known := r.in.grids[name]
	if !r.grids || !known || valid.After(r.now) {
		return ionomaps.Response{Status: replayMissing}, nil
	}
	body, err := os.ReadFile(filepath.Join(r.in.dir, name))
	if err != nil {
		return ionomaps.Response{}, err
	}
	return answer(body), nil
}

// solarDay is a daily row's leading date in the solar file.
var solarDay = regexp.MustCompile(`^(\d{4} \d{2} \d{2})\s`)

// solarBefore is the solar file as it stood at the clock: the rows of days
// before the clock's UTC day, the headers kept.
func solarBefore(body []byte, now time.Time) []byte {
	if len(body) == 0 {
		return nil
	}
	today := now.UTC().Truncate(24 * time.Hour)
	var b bytes.Buffer
	for line := range bytes.Lines(body) {
		if m := solarDay.FindSubmatch(line); m != nil {
			d, err := time.Parse("2006 01 02", string(m[1]))
			if err != nil || !d.Before(today) {
				continue
			}
		}
		b.Write(line)
	}
	return b.Bytes()
}

// askTime is how FastChar takes a time (go-ionomaps `internal/giro`).
const askTime = "2006/01/02 15:04:05"

// recordedFields are a recorded reply's row fields - the time, the score,
// foF2, MUF(3000), M(3000)F2 and hmF2, each but the time and the score with
// its qualifier - as PLAN's fetch asked them (`dry_fetch.py`).
const recordedFields = 10

// errRecordedRow is a recorded row of another shape than PLAN's fetch asked.
var errRecordedRow = fmt.Errorf("a recorded GIRO row is not %d fields: not PLAN's fetch", recordedFields)

// station is a station's reply to go-ionomaps' ask: the recorded rows in
// the window asked and not after the clock, in the columns the ask names -
// the time, the score, foF2 and M(3000)F2 with their qualifiers; missing
// for a station not recorded. The header line naming the requester is left
// out: it is the fetch's, not the station's.
func (r *replay) station(u string) (ionomaps.Response, error) {
	q, err := url.Parse(u)
	if err != nil {
		return ionomaps.Response{}, err
	}
	v := q.Query()
	body, recordedOne := r.in.giro[v.Get("ursiCode")]
	from, errF := time.Parse(askTime, v.Get("fromDate"))
	to, errT := time.Parse(askTime, v.Get("toDate"))
	if errF != nil || errT != nil {
		return ionomaps.Response{}, errUnknownAsk
	}
	if !recordedOne {
		return ionomaps.Response{Status: replayMissing}, nil
	}
	out, err := project(body, from, to, r.now)
	if err != nil {
		return ionomaps.Response{}, err
	}
	return answer(out), nil
}

// project is a recorded reply cut to a window and the clock, in the
// columns go-ionomaps asks.
func project(body []byte, from, to, now time.Time) ([]byte, error) {
	if to.Before(from) {
		return nil, errUnknownAsk
	}
	if len(body) == 0 {
		return nil, nil
	}
	var b bytes.Buffer
	for line := range bytes.Lines(body) {
		text := strings.TrimRight(string(line), "\r\n")
		if strings.HasPrefix(text, "# Requested by") {
			continue
		}
		if strings.HasPrefix(text, "#") || strings.TrimSpace(text) == "" {
			b.WriteString(text + "\n")
			continue
		}
		f := strings.Fields(text)
		if len(f) != recordedFields {
			return nil, errRecordedRow
		}
		at, err := time.Parse("2006-01-02T15:04:05.000Z", f[0])
		if err != nil {
			return nil, errRecordedRow
		}
		if at.Before(from) || at.After(to) || at.After(now) {
			continue
		}
		fmt.Fprintf(&b, "%s %4s %6s %s %5s %s\n", f[0], f[1], f[2], f[3], f[6], f[7])
	}
	return b.Bytes(), nil
}
