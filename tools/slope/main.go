// Command slope is the quality pass's growth statistic (plan §1, §2.1;
// red-team BQ-1, RT-3, R2-1): given a soak's post-GC heap series it
// reports the per-day slope with an autocorrelation-robust confidence
// interval, the 30-day projection of that interval's upper edge, and the
// DETECTION FLOOR — the smallest 30-day growth the run could have
// certified — so a "no growth" verdict is a measurement, not an assertion.
//
// Method. Rows after the warm-up are bucketed per window (default 1 h) and
// the MINIMUM per bucket is the series: it removes in-flight bodies and
// parse transients that a single sample carries. Ordinary least squares
// gives the slope; Newey–West (Bartlett kernel) standard errors account for
// the autocorrelation cache-fill ramps and daily URL churn induce; the
// t-quantile uses n-2 degrees of freedom.
//
// Input: CSV with a header, one row per sample, a UTC timestamp column
// (RFC 3339) and the value column (bytes). scripts/quality/soak.sh writes
// this shape.
//
//	slope -in soak.csv [-col heap_alloc] [-warmup 6h] [-window 1h] [-horizon 720h] [-bar-frac 0.05]
//
// Verdicts and exit codes:
//
//	PASS          0  upper CI × horizon < bar
//	GROWTH        1  the lower CI is above zero and the projection reaches the bar
//	UNCERTIFIABLE 3  the run's detection floor is at or above the bar: it can
//	                 neither pass nor fail the bar — the floor is the bar this
//	                 run can certify (plan §1: restate at the gate, never wave)
//	INSUFFICIENT  2  too few buckets, or bad input
//
// Stdlib only (red-team R2-17).
package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// options are the knobs; every default matches plan §1.
type options struct {
	in      string
	col     string
	tsCol   string
	warmup  time.Duration
	window  time.Duration
	horizon time.Duration
	barFrac float64
}

func run(args []string, out, errOut io.Writer) int {
	var o options
	fs := flag.NewFlagSet("slope", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&o.in, "in", "", "CSV file (required)")
	fs.StringVar(&o.col, "col", "heap_alloc", "value column (bytes)")
	fs.StringVar(&o.tsCol, "ts", "utc", "timestamp column (RFC 3339)")
	fs.DurationVar(&o.warmup, "warmup", 6*time.Hour, "rows before first+warmup are dropped")
	fs.DurationVar(&o.window, "window", time.Hour, "bucket width; the bucket minimum is the series")
	fs.DurationVar(&o.horizon, "horizon", 720*time.Hour, "projection horizon (30 days)")
	fs.Float64Var(&o.barFrac, "bar-frac", 0.05, "pass bar as a fraction of the plateau (median of the series)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if o.in == "" {
		_, _ = fmt.Fprintln(errOut, "slope: -in is required")
		return 2
	}
	f, err := os.Open(o.in)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, "slope:", err)
		return 2
	}
	defer func() { _ = f.Close() }()
	samples, err := readSamples(f, o.tsCol, o.col)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, "slope:", err)
		return 2
	}
	r, err := analyse(samples, o)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, "slope:", err)
		return 2
	}
	_, _ = fmt.Fprint(out, r.String())
	return exitCode(r.Verdict)
}

// exitCode maps a verdict to the process exit code documented above.
func exitCode(verdict string) int {
	switch verdict {
	case "PASS":
		return 0
	case "GROWTH":
		return 1
	case "UNCERTIFIABLE":
		return 3
	}
	return 2
}

// sample is one row: when, and the value in bytes.
type sample struct {
	at    time.Time
	bytes float64
}

// readSamples parses the CSV; rows that do not parse are skipped and
// reported by count, so one bad line never voids a 72-hour run.
func readSamples(r io.Reader, tsCol, col string) ([]sample, error) {
	if err := invariant.Check(tsCol != "" && col != "", "column names must be given"); err != nil {
		return nil, err
	}
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("no header: %w", err)
	}
	ti, vi := columnIndex(header, tsCol), columnIndex(header, col)
	if ti < 0 || vi < 0 {
		return nil, fmt.Errorf("columns %q and %q must both exist in the header %v", tsCol, col, header)
	}
	records, err := cr.ReadAll() // bounded by the file (P10-02); a malformed row is skipped below, not fatal
	if err != nil && len(records) == 0 {
		return nil, fmt.Errorf("unreadable rows: %w", err)
	}
	var out []sample
	for _, rec := range records {
		if len(rec) <= ti || len(rec) <= vi {
			continue
		}
		at, terr := time.Parse(time.RFC3339, rec[ti])
		v, verr := strconv.ParseFloat(rec[vi], 64)
		if terr != nil || verr != nil {
			continue
		}
		out = append(out, sample{at: at, bytes: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].at.Before(out[j].at) })
	return out, nil
}

// columnIndex is the position of name in a CSV header, -1 when absent.
func columnIndex(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

// result is the report.
type result struct {
	N          int     // buckets in the series
	Raw        int     // samples read
	Sigma      float64 // residual standard deviation, MB
	Slope      float64 // MB/day
	SE         float64 // HAC standard error of the slope, MB/day
	Lag        int     // Newey–West lag
	TQ         float64 // t-quantile used
	Lower      float64 // 95 % CI on the slope, MB/day
	Upper      float64
	Projection float64 // upper × horizon, MB
	Floor      float64 // t × SE × horizon: the smallest growth this run could certify, MB
	Plateau    float64 // median of the series, MB
	Bar        float64 // barFrac × plateau, MB
	Horizon    time.Duration
	Verdict    string // PASS | GROWTH | UNCERTIFIABLE | INSUFFICIENT
}

func (r result) String() string {
	days := r.Horizon.Hours() / 24
	return fmt.Sprintf("n=%d buckets (%d samples) sigma=%.2f MB\nslope=%+.3f MB/day  se=%.3f (NW lag %d, t=%.2f)  95%% CI [%+.3f, %+.3f]\nplateau=%.1f MB  bar=%.1f MB (%.0f-day growth)\nprojection(upper)=%.1f MB  floor=%.1f MB\n%s\n",
		r.N, r.Raw, r.Sigma, r.Slope, r.SE, r.Lag, r.TQ, r.Lower, r.Upper, r.Plateau, r.Bar, days, r.Projection, r.Floor, r.Verdict)
}

const mb = 1024 * 1024

// minSeries is the smallest series the statistic is reported for.
const minSeries = 8

// analyse turns samples into the report.
func analyse(samples []sample, o options) (result, error) {
	if err := invariant.Check(o.window > 0 && o.horizon > 0, "window and horizon must be positive"); err != nil {
		return result{}, err
	}
	if err := invariant.Check(o.barFrac > 0 && o.barFrac < 1, "bar-frac must be a fraction of the plateau (0, 1)"); err != nil {
		return result{}, err
	}
	r := result{Raw: len(samples), Horizon: o.horizon}
	if len(samples) == 0 {
		return r, errors.New("no samples parsed")
	}
	xs, ys := bucketMinima(samples, samples[0].at.Add(o.warmup), o.window)
	r.N = len(xs)
	if r.N < minSeries {
		r.Verdict = "INSUFFICIENT"
		return r, fmt.Errorf("only %d buckets after warm-up; need at least %d (%s at %s windows)", r.N, minSeries, o.window*time.Duration(minSeries), o.window)
	}
	slope, intercept := ols(xs, ys)
	resid := make([]float64, r.N)
	for i := range xs {
		resid[i] = ys[i] - (intercept + slope*xs[i])
	}
	r.Slope, r.Sigma = slope, stddev(resid)
	r.Lag = neweyWestLag(r.N)
	r.SE = neweyWestSE(xs, resid, r.Lag)
	r.TQ = tQuantile975(r.N - 2)
	r.Lower, r.Upper = slope-r.TQ*r.SE, slope+r.TQ*r.SE
	days := o.horizon.Hours() / 24
	r.Projection, r.Floor = r.Upper*days, r.TQ*r.SE*days
	r.Plateau = median(ys)
	r.Bar = o.barFrac * r.Plateau
	r.Verdict = verdict(r)
	return r, nil
}

// verdict applies the plan's rule in order: a run that cannot resolve the
// bar says so before it says anything about growth.
func verdict(r result) string {
	if err := invariant.Check(r.Floor >= 0 && r.Bar >= 0, "floor and bar are non-negative sizes"); err != nil {
		return "INSUFFICIENT"
	}
	switch {
	case r.Projection < r.Bar:
		return "PASS"
	case r.Floor >= r.Bar && r.Lower <= 0:
		return "UNCERTIFIABLE"
	default:
		return "GROWTH"
	}
}

// bucketMinima drops samples before start, groups the rest by window and
// returns (days since start, MB) per bucket using each bucket's minimum.
func bucketMinima(samples []sample, start time.Time, window time.Duration) (xs, ys []float64) {
	if err := invariant.Check(window > 0, "bucket window must be positive"); err != nil {
		return nil, nil
	}
	mins := map[int64]float64{}
	for _, s := range samples {
		if s.at.Before(start) {
			continue
		}
		b := int64(s.at.Sub(start) / window)
		if v, ok := mins[b]; !ok || s.bytes < v {
			mins[b] = s.bytes
		}
	}
	keys := make([]int64, 0, len(mins))
	for k := range mins {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	perDay := float64(window) / float64(24*time.Hour)
	for _, k := range keys {
		xs = append(xs, float64(k)*perDay)
		ys = append(ys, mins[k]/mb)
	}
	return xs, ys
}

// ols is the least-squares line through (xs, ys).
func ols(xs, ys []float64) (slope, intercept float64) {
	if err := invariant.Check(len(xs) == len(ys) && len(xs) > 0, "ols needs equal, non-empty x and y"); err != nil {
		return 0, 0
	}
	mx, my := mean(xs), mean(ys)
	var sxy, sxx float64
	for i := range xs {
		sxy += (xs[i] - mx) * (ys[i] - my)
		sxx += (xs[i] - mx) * (xs[i] - mx)
	}
	if sxx == 0 {
		return 0, my
	}
	slope = sxy / sxx
	return slope, my - slope*mx
}

// neweyWestLag is the common bandwidth rule floor(4·(n/100)^(2/9)).
func neweyWestLag(n int) int {
	if err := invariant.Check(n > 0, "lag rule needs n > 0"); err != nil {
		return 0
	}
	return int(math.Floor(4 * math.Pow(float64(n)/100, 2.0/9)))
}

// neweyWestSE is the HAC standard error of the OLS slope with a Bartlett
// kernel: Var(β) = S / Sxx², S = Σ_{|k|≤L} w_k Σ_t u_t u_{t-k} x̃_t x̃_{t-k}.
func neweyWestSE(xs, resid []float64, lag int) float64 {
	n := len(xs)
	if err := invariant.Check(len(resid) == n && n > 2, "HAC needs residuals per x and n > 2"); err != nil {
		return 0
	}
	if err := invariant.Check(lag >= 0, "Newey-West lag must be non-negative"); err != nil {
		return 0
	}
	mx := mean(xs)
	xt := make([]float64, n)
	var sxx float64
	for i := range xs {
		xt[i] = xs[i] - mx
		sxx += xt[i] * xt[i]
	}
	if sxx == 0 {
		return 0
	}
	var s float64
	for k := 0; k <= lag && k < n; k++ {
		w := 1 - float64(k)/float64(lag+1)
		var g float64
		for t := k; t < n; t++ {
			g += resid[t] * xt[t] * resid[t-k] * xt[t-k]
		}
		if k == 0 {
			s += g
		} else {
			s += 2 * w * g
		}
	}
	if s < 0 {
		s = 0
	}
	// Small-sample correction n/(n-2), as regression packages apply.
	return math.Sqrt(s*float64(n)/float64(n-2)) / sxx
}

// tQuantile975 is the two-sided 95 % Student-t quantile for df degrees of
// freedom (embedded table; linear in 1/df between rows; z beyond 200).
func tQuantile975(df int) float64 {
	table := []struct {
		df int
		q  float64
	}{{1, 12.706}, {2, 4.303}, {3, 3.182}, {4, 2.776}, {5, 2.571}, {6, 2.447}, {7, 2.365}, {8, 2.306}, {9, 2.262}, {10, 2.228},
		{12, 2.179}, {15, 2.131}, {20, 2.086}, {25, 2.060}, {30, 2.042}, {40, 2.021}, {60, 2.000}, {80, 1.990}, {120, 1.980}, {200, 1.972}}
	if err := invariant.Check(df >= 1, "t-quantile needs at least one degree of freedom"); err != nil {
		return table[0].q
	}
	if df >= 200 {
		return 1.960
	}
	for i := 1; i < len(table); i++ {
		if df <= table[i].df {
			a, b := table[i-1], table[i]
			// interpolate in 1/df, where the quantile is close to linear
			fa, fb, fd := 1/float64(a.df), 1/float64(b.df), 1/float64(df)
			return b.q + (a.q-b.q)*(fd-fb)/(fa-fb)
		}
	}
	return 1.960
}

func mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var s float64
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

func stddev(v []float64) float64 {
	if len(v) < 2 {
		return 0
	}
	m := mean(v)
	var s float64
	for _, x := range v {
		s += (x - m) * (x - m)
	}
	return math.Sqrt(s / float64(len(v)-1))
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	if len(c)%2 == 1 {
		return c[len(c)/2]
	}
	return (c[len(c)/2-1] + c[len(c)/2]) / 2
}
