// Command nwrtable turns the NWS county-coverage list (CCL.js — the
// public-domain JS arrays behind weather.gov/nwr/county_coverage) into the
// vendored transmitter table domains/radio/stream/transmitters.csv
// (architecture §10.6, B4). One row per transmitter: callsign, site,
// state, frequency, lat, lon, power (W), status, and the SAME codes it
// covers ("|"-joined). Refresh per release:
//
//	go run ./tools/nwrtable -in CCL.js -out domains/radio/stream/transmitters.csv
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	in := flag.String("in", "", "CCL.js as served by weather.gov/source/nwr/JS/CCL.js")
	out := flag.String("out", "domains/radio/stream/transmitters.csv", "output CSV")
	flag.Parse()
	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rows, err := Parse(string(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"callsign", "site", "state", "freq_mhz", "lat", "lon", "power_w", "status", "same"})
	for _, r := range rows {
		_ = w.Write([]string{r.Callsign, r.Site, r.State, r.Freq, r.Lat, r.Lon, r.Power, r.Status, strings.Join(r.SAME, "|")})
	}
	w.Flush()
	fmt.Fprintf(os.Stderr, "nwrtable: %d transmitters -> %s\n", len(rows), *out)
}

// Transmitter is one NWR station with the SAME codes it covers.
type Transmitter struct {
	Callsign, Site, State, Freq, Lat, Lon, Power, Status string
	SAME                                                 []string
}

var assign = regexp.MustCompile(`^(\w+)\[(\d+)\] = "(.*)";$`)

// Parse folds CCL.js's parallel arrays (one entry per county×transmitter)
// into transmitters keyed by callsign, SAME codes accumulated and sorted.
func Parse(js string) ([]Transmitter, error) {
	cols := map[string]map[int]string{}
	max := -1
	for _, line := range strings.Split(js, "\n") {
		m := assign.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		i, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		if cols[m[1]] == nil {
			cols[m[1]] = map[int]string{}
		}
		cols[m[1]][i] = m[3]
		if i > max {
			max = i
		}
	}
	if max < 0 || cols["CALLSIGN"] == nil {
		return nil, fmt.Errorf("nwrtable: no CCL rows found")
	}
	byCall := map[string]*Transmitter{}
	for i := 0; i <= max; i++ {
		call := strings.TrimSpace(cols["CALLSIGN"][i])
		if call == "" || cols["LAT"][i] == "" || cols["LON"][i] == "" {
			continue // "No Coverage" county rows
		}
		t := byCall[call]
		if t == nil {
			t = &Transmitter{Callsign: call, Site: cols["SITENAME"][i], State: cols["SITESTATE"][i], Freq: cols["FREQ"][i],
				Lat: cols["LAT"][i], Lon: cols["LON"][i], Power: cols["PWR"][i], Status: cols["STATUS"][i]}
			byCall[call] = t
		}
		if same := cols["SAME"][i]; same != "" {
			t.SAME = append(t.SAME, same)
		}
	}
	out := make([]Transmitter, 0, len(byCall))
	for _, t := range byCall {
		sort.Strings(t.SAME)
		t.SAME = dedupe(t.SAME)
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Callsign < out[j].Callsign })
	return out, nil
}

func dedupe(s []string) []string {
	out := s[:0]
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}
