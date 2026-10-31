package main

// spots.go - the WSPR spots: the one query a day wspr.live is sent, the
// fetch under its courtesy, and the rows read back.

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/propanswers"
)

// wsprLive is wspr.live's public query address: ClickHouse over HTTP.
const wsprLive = "https://db1.wspr.live/"

// userAgent names the instrument to wspr.live.
const userAgent = "watchpost-reachscore (+https://github.com/branden-thompson/watchpost)"

// The courtesy wspr.live is asked with: one query at a time, a pause
// between, and after a refusal or no answer 60 s doubling to 15 minutes,
// then the day given up - the shape D-39 gives GIRO.
const (
	queryGap     = 15 * time.Second
	firstBackOff = time.Minute
	longestPause = 15 * time.Minute
	statusOK     = 200
)

// usSouth to usEast bound the mainland US as PLAN's queries did
// (`dry-run.md`, "Requests made").
const (
	usSouth, usNorth = 24.0, 50.0
	usWest, usEast   = -125.0, -66.0
)

// inUS reports whether a place is in the mainland US box.
func inUS(p ionomaps.LatLon) bool {
	return p.Lat >= usSouth && p.Lat <= usNorth && p.Lon >= usWest && p.Lon <= usEast
}

// wsprBands are wspr.live's band numbers - the MHz, truncated - for the
// bands go-ionomaps answers, in its order.
var wsprBands = []struct {
	n    int
	band ionomaps.Band
}{
	{1, ionomaps.Band160m}, {3, ionomaps.Band80m}, {5, ionomaps.Band60m}, {7, ionomaps.Band40m}, {10, ionomaps.Band30m},
	{14, ionomaps.Band20m}, {18, ionomaps.Band17m}, {21, ionomaps.Band15m}, {24, ionomaps.Band12m}, {28, ionomaps.Band10m},
}

// bandOf is go-ionomaps' band for a wspr.live band number; false for a
// band it does not answer.
func bandOf(n int) (ionomaps.Band, bool) {
	for _, b := range wsprBands {
		if b.n == n {
			return b.band, true
		}
	}
	return 0, false
}

// spotsQuery is the day's one query: the spots heard in the mainland US on
// the answered bands from a transmitter in the US or near a continent's
// reference point (FR-1.9), aggregated to each pair of locators, band and
// hour - the count and the strongest SNR normalised to 100 W
// (`snr + (50 - power)`, D-73). Locators, never callsigns: the instrument
// needs where a station is, not who. The day reaches back to 23:00 the day
// before, its first window's hour, and stops at 23:00, the next day's.
func spotsQuery(day time.Time, regionKm float64) string {
	var bands, near []string
	for _, b := range wsprBands {
		bands = append(bands, strconv.Itoa(b.n))
	}
	for _, c := range propanswers.Continents {
		near = append(near, fmt.Sprintf("greatCircleDistance(tx_lon, tx_lat, %.2f, %.2f) < %.0f", c.At.Lon, c.At.Lat, regionKm*1000))
	}
	from, to := day.Add(-time.Hour).Format(time.DateTime), day.Add(23*time.Hour).Format(time.DateTime)
	return "SELECT tx_loc, tx_lat, tx_lon, rx_loc, rx_lat, rx_lon, band, toStartOfHour(time) AS hour, count() AS n, max(snr + (50 - power)) AS best" +
		" FROM wspr.rx WHERE time >= '" + from + "' AND time < '" + to + "'" +
		" AND band IN (" + strings.Join(bands, ",") + ")" +
		fmt.Sprintf(" AND rx_lat BETWEEN %g AND %g AND rx_lon BETWEEN %g AND %g", usSouth, usNorth, usWest, usEast) +
		" AND tx_loc != rx_loc" +
		fmt.Sprintf(" AND ((tx_lat BETWEEN %g AND %g AND tx_lon BETWEEN %g AND %g) OR ", usSouth, usNorth, usWest, usEast) +
		strings.Join(near, " OR ") + ")" +
		" GROUP BY tx_loc, tx_lat, tx_lon, rx_loc, rx_lat, rx_lon, band, hour ORDER BY hour, band, tx_loc, rx_loc FORMAT TSV"
}

// spotsFile is where a day's spots are kept in the output folder.
func spotsFile(dir string, day time.Time) string {
	return filepath.Join(dir, "wspr-"+day.Format(time.DateOnly)+".tsv")
}

// fetcher asks wspr.live: the address, the client, and the pause, a test's
// to replace.
type fetcher struct {
	base   string
	client *http.Client
	sleep  func(time.Duration)
	log    io.Writer
}

// spotsCommand fetches each day's spots not yet kept, one query at a time.
func spotsCommand(args []string, out io.Writer) (err error) {
	fs := flag.NewFlagSet("spots", flag.ContinueOnError)
	from := fs.String("from", "", "first UTC day, YYYY-MM-DD")
	to := fs.String("to", "", "last UTC day, YYYY-MM-DD")
	dir := fs.String("out", "", "folder outside the tree the spots are kept in")
	regionKm := fs.Float64("region-km", defaultRegionKm, "a continent's region: transmitters within this distance of its reference point")
	if err = fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return errUsage
	}
	ds, err := days(*from, *to)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(*dir, 0o700); err != nil {
		return err
	}
	logf, err := os.OpenFile(filepath.Join(*dir, "requests.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := logf.Close(); err == nil {
			err = cerr // the request log is the record of what was sent
		}
	}()
	f := fetcher{base: wsprLive, client: &http.Client{Timeout: 2 * time.Minute}, sleep: time.Sleep, log: logf}
	return f.fetchDays(ds, *dir, *regionKm, out)
}

// fetchDays fetches each day not yet kept, pausing between asks.
func (f fetcher) fetchDays(ds []time.Time, dir string, regionKm float64, out io.Writer) error {
	if !(regionKm > 0) {
		return errUsage // no region: the query would ask for every transmitter in the US alone
	}
	asked := false
	for _, d := range ds {
		path := spotsFile(dir, d)
		if _, err := os.Stat(path); err == nil {
			if _, err := fmt.Fprintf(out, "%s kept already: %s\n", d.Format(time.DateOnly), path); err != nil {
				return err
			}
			continue
		}
		if asked {
			f.sleep(queryGap)
		}
		asked = true
		body, err := f.fetchDay(d, regionKm)
		if err != nil {
			return fmt.Errorf("%s: %w", d.Format(time.DateOnly), err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "%s: %d bytes -> %s\n", d.Format(time.DateOnly), len(body), path); err != nil {
			return err
		}
	}
	return nil
}

// errGivenUp is a day wspr.live refused or left unanswered through the
// longest pause.
var errGivenUp = fmt.Errorf("wspr.live refused or did not answer through the %v pause; given up", longestPause)

// fetchDay asks for a day, backing off after a refusal or no answer.
func (f fetcher) fetchDay(day time.Time, regionKm float64) ([]byte, error) {
	u := f.base + "?query=" + url.QueryEscape(spotsQuery(day, regionKm))
	for wait := firstBackOff; wait <= longestPause; wait *= 2 {
		body, status, err := f.ask(u, day)
		if err == nil && status == statusOK {
			return body, nil
		}
		if err == nil && status != http.StatusTooManyRequests && status < 500 {
			return nil, fmt.Errorf("wspr.live answered %d", status)
		}
		f.sleep(wait)
	}
	return nil, errGivenUp
}

// rateHeaders are the only headers kept from an answer (D-83): never a
// whole header.
var rateHeaders = []string{"retry-after", "x-ratelimit", "ratelimit"}

// ask is one request, logged: when, the day, the status, the size, the
// seconds and the rate headers alone.
func (f fetcher) ask(u string, day time.Time) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	start := time.Now()
	resp, err := f.client.Do(req)
	if err != nil {
		if _, lerr := fmt.Fprintf(f.log, "%s %s no answer %.1fs\n", start.UTC().Format(time.RFC3339), day.Format(time.DateOnly), time.Since(start).Seconds()); lerr != nil {
			return nil, 0, lerr // the request log is the record of what was sent
		}
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }() // read in full below; a failed close loses nothing
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSpotsBytes+1))
	if err == nil && len(body) > maxSpotsBytes {
		err = errTooLarge
	}
	var rate []string
	for k, v := range resp.Header {
		for _, keep := range rateHeaders {
			if strings.HasPrefix(strings.ToLower(k), keep) {
				rate = append(rate, k+"="+strings.Join(v, ","))
			}
		}
	}
	if _, lerr := fmt.Fprintf(f.log, "%s %s %d %d bytes %.1fs %s\n", start.UTC().Format(time.RFC3339), day.Format(time.DateOnly), resp.StatusCode, len(body), time.Since(start).Seconds(), strings.Join(rate, " ")); lerr != nil {
		return nil, 0, lerr
	}
	return body, resp.StatusCode, err
}

// maxSpotsBytes is the largest day's answer kept: a day's aggregate is
// tens of megabytes; past this it is not the query's.
const maxSpotsBytes = 512 << 20

// errTooLarge is an answer larger than any day's aggregate.
var errTooLarge = fmt.Errorf("wspr.live's answer is over %d bytes: not the day's aggregate", maxSpotsBytes)

// spot is one aggregate row: a pair of locators heard on a band in an hour,
// how often, and the strongest SNR normalised to 100 W.
type spot struct {
	Tx, Rx     string
	TxAt, RxAt ionomaps.LatLon
	Band       ionomaps.Band
	Hour       time.Time
	N          int
	Best       float64
}

// minBest and maxBest bound a normalised SNR: WSPR decodes from about -34
// dB, at powers from 0 to 60 dBm, so normalised to 100 W a real spot lies
// well inside them.
const (
	minBest = -100.0
	maxBest = 100.0
)

// spotFields are a row's fields, as spotsQuery selects them.
const spotFields = 10

// readSpots reads a day's rows; a row of another shape, or a value out of
// its bounds, is an error naming its line.
func readSpots(r io.Reader) ([]spot, error) {
	var out []spot
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for line := 1; line <= maxSpotRows+1 && sc.Scan(); line++ {
		if line > maxSpotRows {
			return nil, errTooMany
		}
		s, err := readSpot(sc.Text())
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		out = append(out, s)
	}
	return out, sc.Err()
}

// maxSpotRows is the most rows a day's spots are read to: a day's
// aggregate is a few hundred thousand.
const maxSpotRows = 20_000_000

// errTooMany is a day's file longer than any day's aggregate.
var errTooMany = fmt.Errorf("a day's spots run past %d rows: not the query's", maxSpotRows)

// errRow is a row that is not the query's.
var errRow = fmt.Errorf("not a row of the spots query: %d tab-separated fields, places on the globe, a band go-ionomaps answers", spotFields)

// readSpot reads one row.
func readSpot(text string) (spot, error) {
	f := strings.Split(text, "\t")
	if len(f) != spotFields {
		return spot{}, errRow
	}
	txLat, e1 := strconv.ParseFloat(f[1], 64)
	txLon, e2 := strconv.ParseFloat(f[2], 64)
	rxLat, e3 := strconv.ParseFloat(f[4], 64)
	rxLon, e4 := strconv.ParseFloat(f[5], 64)
	bn, e5 := strconv.Atoi(f[6])
	hour, e6 := time.Parse(time.DateTime, f[7])
	n, e7 := strconv.Atoi(f[8])
	best, e8 := strconv.ParseFloat(f[9], 64)
	band, known := bandOf(bn)
	parsed := e1 == nil && e2 == nil && e3 == nil && e4 == nil && e5 == nil && e6 == nil && e7 == nil && e8 == nil
	if !parsed || !known || n < 1 {
		return spot{}, errRow
	}
	if best < minBest || best > maxBest {
		return spot{}, errRow
	}
	tx, rx := ionomaps.LatLon{Lat: txLat, Lon: txLon}, ionomaps.LatLon{Lat: rxLat, Lon: rxLon}
	if !onGlobe(tx) || !onGlobe(rx) || f[0] == "" || f[3] == "" {
		return spot{}, errRow
	}
	return spot{Tx: f[0], Rx: f[3], TxAt: tx, RxAt: rx, Band: band, Hour: hour, N: n, Best: best}, nil
}

// onGlobe reports whether a place is a latitude and longitude.
func onGlobe(p ionomaps.LatLon) bool {
	return p.Lat >= -90 && p.Lat <= 90 && p.Lon >= -180 && p.Lon <= 180
}
