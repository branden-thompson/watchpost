package main

// interval.go - the 95 % paired bootstrap interval the graded run shows
// beside each agreement and each lead (D-228); D-99's target is still
// judged on the point estimate.

import (
	"math/rand/v2"
	"slices"
)

// bootReps is how many resamples the bootstrap draws.
const bootReps = 2000

// interval is a 95 % interval, in percentage points.
type interval struct{ lo, hi float64 }

// bounds are a group's intervals: each column's agreement, and the shipped
// build's lead over the climatology.
type bounds struct {
	cols [columns]interval
	lead interval
}

// intervals are each metric's bands' bounds and each metric's whole.
type intervals struct {
	byBand map[base]bounds
	all    [metrics]bounds
}

// bootstrap is the paired bootstrap of a group of scenarios: drawn with
// replacement bootReps times, each column's agreement and the shipped
// build's lead recomputed on the same draw - so the lead's interval keeps
// the pairing - and the 2.5th and 97.5th percentiles of each kept.
func bootstrap(agree [][columns]bool, rng *rand.Rand) bounds {
	var out bounds
	n := len(agree)
	if n == 0 {
		return out
	}
	var stats [columns + 1][]float64
	for range bootReps {
		var hits [columns]int
		for range n {
			a := agree[rng.IntN(n)]
			for c := range columns {
				if a[c] {
					hits[c]++
				}
			}
		}
		for c := range columns {
			stats[c] = append(stats[c], 100*float64(hits[c])/float64(n))
		}
		stats[columns] = append(stats[columns], 100*float64(hits[shipped]-hits[climatology])/float64(n))
	}
	pick := func(s []float64) interval {
		slices.Sort(s)
		return interval{lo: s[bootReps*25/1000], hi: s[bootReps*975/1000-1]}
	}
	for c := range columns {
		out.cols[c] = pick(stats[c])
	}
	out.lead = pick(stats[columns])
	return out
}

// intervalsOf are the drawn set's intervals, each band's and each metric's,
// in a fixed order so the seed fixes them.
func intervalsOf(set []scenario, answers [][columns]bool, seed uint64) intervals {
	out := intervals{byBand: map[base]bounds{}}
	if len(answers) != len(set) {
		return out
	}
	rng := rand.New(rand.NewPCG(seed, seed^0x51ed270b27b5f9c3))
	groups := map[base][][columns]bool{}
	var whole [metrics][][columns]bool
	for i, s := range set {
		var a [columns]bool
		for c := range columns {
			a[c] = answers[i][c] == (s.Outcome == open)
		}
		k := base{s.Metric, s.Band}
		groups[k] = append(groups[k], a)
		whole[s.Metric] = append(whole[s.Metric], a)
	}
	for m := range metrics {
		for _, wb := range wsprBands {
			if g := groups[base{m, wb.band}]; len(g) > 0 {
				out.byBand[base{m, wb.band}] = bootstrap(g, rng)
			}
		}
		out.all[m] = bootstrap(whole[m], rng)
	}
	return out
}
