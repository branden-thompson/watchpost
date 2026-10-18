// Command reachscore is the M1 and M4 instrument (0.19.0 W10.3; D-208,
// D-218 to D-221; `07-readiness/m1-m4-instrument.md`). "spots" fetches the
// days' WSPR spots heard in the mainland US from wspr.live, one filtered
// aggregate query a day; "score" draws the scenarios from them (B1), answers
// each three ways through go-ionomaps by replaying the days' recorded NOAA
// and GIRO files - the climatology, the background alone, the shipped build
// - reads each answer as the product does (C1), scores it against the spots
// (A2) and prints the table by band. The spots, the recorded files and the
// scenarios stay outside the tree: GIRO's terms, and the locators place
// stations.
//
//	go run ./tools/reachscore spots -from 2026-09-29 -to 2026-10-05 -out DIR
//	go run ./tools/reachscore score -inputs DIR -solar FILE -spots DIR -from 2026-09-29 -to 2026-10-05 -out DIR
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	err := run(os.Args[1:], os.Stdout)
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "reachscore:", err)
	os.Exit(1)
}

// errUsage is a command line naming no command the tool has.
var errUsage = errors.New("usage: reachscore spots|score [flags]; -h after either lists its flags")

// run is one command, named by its first argument.
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errUsage
	}
	if out == nil {
		return errUsage
	}
	switch args[0] {
	case "spots":
		return spotsCommand(args[1:], out)
	case "score":
		return scoreCommand(args[1:], out)
	}
	return errUsage
}

// maxDays is the most days one run covers: a fortnight and more is not a
// dry run, and wspr.live is asked once a day.
const maxDays = 31

// errDays is a span of days that does not read, runs backwards or is
// longer than maxDays.
var errDays = fmt.Errorf("-from and -to are UTC days, YYYY-MM-DD, -to not before -from, at most %d days", maxDays)

// days are the UTC days from and to, both included.
func days(from, to string) ([]time.Time, error) {
	if from == "" || to == "" {
		return nil, errDays
	}
	a, errA := time.Parse(time.DateOnly, from)
	b, errB := time.Parse(time.DateOnly, to)
	if errA != nil || errB != nil {
		return nil, errDays
	}
	n := int(b.Sub(a)/(24*time.Hour)) + 1
	if n < 1 || n > maxDays {
		return nil, errDays
	}
	out := make([]time.Time, n)
	for i := range out {
		out[i] = a.AddDate(0, 0, i)
	}
	return out, nil
}
