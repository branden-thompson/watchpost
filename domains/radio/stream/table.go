// Package stream resolves a location to NOAA Weather Radio audio (B4,
// architecture §5/§10.6, AI-4): the vendored NWS transmitter table says
// which transmitter covers a county and where every transmitter stands;
// two community Icecast directories (wxradio.org, weatherUSA) say which
// transmitters are actually relayed. The covering transmitter plays when
// it is relayed; otherwise the nearest relayed transmitter does, labelled
// with its distance so nobody mistakes a neighbour's broadcast for their own.
package stream

import (
	_ "embed"
	"encoding/csv"
	"sort"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// TableAttribution credits the vendored table's source.
const TableAttribution = "NOAA NWR transmitter list (weather.gov/nwr)" // public domain

//go:embed transmitters.csv
var transmittersCSV string

// Transmitter is one NWR station from the vendored table.
type Transmitter struct {
	Callsign string
	Site     string
	State    string
	FreqMHz  string
	Lat, Lon float64
	PowerW   int
	Status   string   // NORMAL | DEGRADED | OUT OF SERVICE
	SAME     []string // covered county SAME codes ("0" + FIPS)
}

// Table is the parsed transmitter table.
type Table struct {
	byCall map[string]*Transmitter
	bySAME map[string][]*Transmitter
	all    []*Transmitter
}

// LoadTable parses the embedded table (~107 KB; a few ms — each Resolver
// owns its copy rather than a package global).
func LoadTable() (*Table, error) { return parseTable(transmittersCSV) }

func parseTable(raw string) (*Table, error) {
	recs, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	if err != nil {
		return nil, err
	}
	if err := invariant.Check(len(recs) > 1 && len(recs[0]) == 9, "transmitter table: unexpected shape"); err != nil {
		return nil, err
	}
	t := &Table{byCall: map[string]*Transmitter{}, bySAME: map[string][]*Transmitter{}}
	for _, r := range recs[1:] {
		lat, err1 := strconv.ParseFloat(r[4], 64)
		lon, err2 := strconv.ParseFloat(r[5], 64)
		if err1 != nil || err2 != nil || r[0] == "" {
			continue // a row without a position cannot be "nearest"
		}
		pw, _ := strconv.Atoi(r[6])
		tx := &Transmitter{Callsign: r[0], Site: r[1], State: r[2], FreqMHz: r[3], Lat: lat, Lon: lon, PowerW: pw, Status: r[7]}
		if r[8] != "" {
			tx.SAME = strings.Split(r[8], "|")
		}
		t.byCall[tx.Callsign] = tx
		t.all = append(t.all, tx)
		for _, s := range tx.SAME {
			t.bySAME[s] = append(t.bySAME[s], tx)
		}
	}
	return t, nil
}

// Len is the number of transmitters with a position.
func (t *Table) Len() int { return len(t.all) }

// ByCallsign looks a transmitter up.
func (t *Table) ByCallsign(call string) (*Transmitter, bool) {
	tx, ok := t.byCall[strings.ToUpper(call)]
	return tx, ok
}

// Covering lists the transmitters whose coverage includes a SAME code.
func (t *Table) Covering(same string) []*Transmitter { return t.bySAME[same] }

// Near is a transmitter with its distance from a point.
type Near struct {
	*Transmitter
	KM float64
}

// Nearest lists the n closest transmitters to a point (any status).
func (t *Table) Nearest(lat, lon float64, n int) []Near {
	out := make([]Near, 0, len(t.all))
	for _, tx := range t.all {
		out = append(out, Near{tx, geo.HaversineKM(lat, lon, tx.Lat, tx.Lon)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].KM < out[j].KM })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// SAMEFromFIPS turns a county FIPS ("06073") or a NWS county UGC ("CAC073"
// with a state FIPS) into the table's SAME form ("006073").
func SAMEFromFIPS(fips string) string {
	fips = strings.TrimSpace(fips)
	if len(fips) == 5 {
		return "0" + fips
	}
	if len(fips) == 6 {
		return fips
	}
	return ""
}

// SAMEFromUGC turns a NWS county UGC ("CAC073") into a SAME code
// ("006073"): "0" + state FIPS + county number.
func SAMEFromUGC(ugc string) string {
	ugc = strings.ToUpper(strings.TrimSpace(ugc))
	if len(ugc) != 6 || ugc[2] != 'C' {
		return ""
	}
	fips, ok := stateFIPS()[ugc[:2]]
	if !ok {
		return ""
	}
	return "0" + fips + ugc[3:]
}

// stateFIPS is the FIPS 5-2 state/territory table (data, not code).
func stateFIPS() map[string]string {
	return map[string]string{
		"AL": "01", "AK": "02", "AZ": "04", "AR": "05", "CA": "06", "CO": "08", "CT": "09", "DE": "10", "DC": "11",
		"FL": "12", "GA": "13", "HI": "15", "ID": "16", "IL": "17", "IN": "18", "IA": "19", "KS": "20", "KY": "21",
		"LA": "22", "ME": "23", "MD": "24", "MA": "25", "MI": "26", "MN": "27", "MS": "28", "MO": "29", "MT": "30",
		"NE": "31", "NV": "32", "NH": "33", "NJ": "34", "NM": "35", "NY": "36", "NC": "37", "ND": "38", "OH": "39",
		"OK": "40", "OR": "41", "PA": "42", "RI": "44", "SC": "45", "SD": "46", "TN": "47", "TX": "48", "UT": "49",
		"VT": "50", "VA": "51", "WA": "53", "WV": "54", "WI": "55", "WY": "56", "AS": "60", "GU": "66", "MP": "69",
		"PR": "72", "VI": "78",
	}
}
