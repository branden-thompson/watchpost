package history

// fuzz_test.go — the store reads files it did not write: another instance's,
// another version's, a damaged disk's. The day and year documents are fuzzed
// from ones the store writes: no panic; a day's records are its own hours, in
// order and no more than its step allows; a year's days are the ones asked
// for; no value infinite.

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// written is the document a store wrote for one record, as JSON.
func written(f *testing.F) []byte {
	f.Helper()
	dir, now := f.TempDir(), t0
	s := Open(dir, func() time.Time { return now }, grid)
	if !s.Put(grid.Name, rec(t0, 0, 21.04, math.NaN(), -3.96, 10)) {
		f.Fatal("not written")
	}
	raw, err := os.ReadFile(filesAt(filepath.Join(dir, grid.Name, "v1", "ndfd", "us-a"), t0).bucket)
	if err != nil {
		f.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		f.Fatal(err)
	}
	body, err := io.ReadAll(zr)
	if err != nil {
		f.Fatal(err)
	}
	return body
}

// writeDoc writes a document's body, compressed, where the store reads it.
func writeDoc(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(body)
	_ = zw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// noInf fails when a field's values hold an infinite one.
func noInf(t *testing.T, fields map[string][]float64) {
	t.Helper()
	for name, vs := range fields {
		for _, v := range vs {
			if math.IsInf(v, 0) {
				t.Fatalf("%s holds %v", name, v)
			}
		}
	}
}

func FuzzReadDay(f *testing.F) {
	one := written(f)
	f.Add(one)
	var doc dayDoc
	if err := json.Unmarshal(one, &doc); err != nil {
		f.Fatal(err)
	}
	second := doc.Hours[0]
	second.Hour = second.Hour.Add(time.Hour)
	doc.Hours = append(doc.Hours, second)
	two, err := json.Marshal(doc)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(two)
	year, err := json.Marshal(yearDoc{Schema: schema, Dataset: grid.Name, Version: grid.Version, Key: ndfd,
		Days: []dayEntry{{Date: "2026-09-30", Shape: box, Hours: 2, Stats: map[string]map[string][]*float64{"temp": {"min": second.Values["temp"]}}}}})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(year)
	day := time.Date(t0.Year(), t0.Month(), t0.Day(), 0, 0, 0, 0, time.UTC)
	dir := f.TempDir() // one a worker, its files written over by each input
	f.Fuzz(func(t *testing.T, body []byte) {
		now := t0
		files := filesAt(filepath.Join(dir, grid.Name, "v1", "ndfd", "us-a"), t0)
		writeDoc(t, files.day, body)
		writeDoc(t, files.bucket, body)
		writeDoc(t, files.year, body)
		s := Open(dir, func() time.Time { return now }, grid)
		got := s.Range(grid.Name, ndfd, day, day.Add(24*time.Hour-time.Second), 1000)
		if len(got) > 24 {
			t.Fatalf("%d hours read for a day of hourly records", len(got))
		}
		for i, r := range got {
			if r.At.Before(day) || !r.At.Before(day.Add(24*time.Hour)) {
				t.Fatalf("a record at %v read for the day of %v", r.At, day)
			}
			if i > 0 && !got[i-1].At.Before(r.At) {
				t.Fatalf("records out of order: %v then %v", got[i-1].At, r.At)
			}
			noInf(t, r.Values)
		}
		_, _ = s.Get(grid.Name, ndfd, t0)
		from := day.AddDate(0, 0, -3)
		days := s.Days(grid.Name, ndfd, from, day, 10)
		if len(days) > 10 {
			t.Fatalf("%d days read for at most 10", len(days))
		}
		for _, d := range days {
			if d.Date.Before(from) || d.Date.After(day) {
				t.Fatalf("a day of %v read for %v to %v", d.Date, from, day)
			}
			noInf(t, d.Min)
			noInf(t, d.Max)
			noInf(t, d.Mean)
		}
	})
}
