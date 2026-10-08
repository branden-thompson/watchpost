package history

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

// before is values as 0.18.0 held them: a pointer a value, nil when missing.
func before(v values) []*float64 {
	if v == nil {
		return nil
	}
	out := make([]*float64, len(v))
	for i, x := range v {
		if !math.IsNaN(x) {
			out[i] = &x
		}
	}
	return out
}

// VALUES ARE WRITTEN AS BEFORE (FR-6.5): held compactly, they are the same
// bytes on disk as 0.18.0's pointer a value, so every file already written is
// read, and every roll-up part's size is what the budget measured.
func TestValuesAreWrittenAsBefore(t *testing.T) {
	cases := []values{
		nil,
		{},
		{math.NaN()},
		{0, -0.5, 1, 12.25, -273.15, 1e-7, 1.5e-9, 2.5e21, 1e20, math.NaN(), 123456789.123},
	}
	for _, v := range cases {
		got, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		want, err := json.Marshal(before(v))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%v written as %s; 0.18.0 wrote %s", []float64(v), got, want)
		}
	}
}

// VALUES ARE READ AS BEFORE: whatever 0.18.0's pointers read, values read the
// same, a null as NaN; whatever they refused, values refuse.
func FuzzValuesAreReadAsBefore(f *testing.F) {
	for _, seed := range []string{`null`, `[]`, `[ ]`, `[1,null,-2.5]`, `[ 1 , null ]`, `[1e400]`, `[1e-400]`,
		`["1"]`, `[true]`, `[[1]]`, `[{"a":1,"b":2}]`, `{"a":1}`, `"x"`, `7`, `[0.1e2,-0E-3]`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		var old []*float64
		oldErr := json.Unmarshal(raw, &old)
		var got values
		err := json.Unmarshal(raw, &got)
		if (err == nil) != (oldErr == nil) {
			t.Fatalf("%q: read with %v; 0.18.0 read it with %v", raw, err, oldErr)
		}
		if err != nil {
			return
		}
		if (got == nil) != (old == nil) || len(got) != len(old) {
			t.Fatalf("%q: read as %v; 0.18.0 read %d value(s), nil %v", raw, []float64(got), len(old), old == nil)
		}
		for i, p := range old {
			if (p == nil) != math.IsNaN(got[i]) || (p != nil && *p != got[i]) {
				t.Fatalf("%q: value %d read as %v", raw, i, got[i])
			}
		}
	})
}

// valuesAllocBudget is what reading a field's values may allocate, whatever
// their number: 0.18.0 allocated one a value.
const valuesAllocBudget = 2

// A FIELD'S VALUES ARE ONE ALLOCATION, NOT ONE A VALUE (FR-6.5, PF-F5): a day
// of a global grid held 12.5 to 50 MB in pointers.
func TestValuesReadAllocBudget(t *testing.T) {
	v := make(values, 4096)
	for i := range v {
		v[i] = float64(i) / 4
		if i%7 == 0 {
			v[i] = math.NaN()
		}
	}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var back values
	got := testing.AllocsPerRun(20, func() {
		if err := back.UnmarshalJSON(raw); err != nil {
			t.Fatal(err)
		}
	})
	t.Logf("%d values read in %.0f allocations (budget %d)", len(v), got, valuesAllocBudget)
	if got > valuesAllocBudget {
		t.Errorf("%d values read in %.0f allocations; the budget is %d", len(v), got, valuesAllocBudget)
	}
}

// A RECORD READ IS ITS READER'S OWN: the store holds a day's values as read,
// and a reader that changes what it was given changes nothing the store holds.
func TestARecordReadIsItsReadersOwn(t *testing.T) {
	dir, now := t.TempDir(), t0
	s := Open(dir, func() time.Time { return now }, grid)
	s.Put(grid.Name, rec(t0, 0, 1, 2, 3, 4))
	r, ok := s.Get(grid.Name, ndfd, t0)
	if !ok {
		t.Fatal("the record was not read")
	}
	r.Values["temp"][0] = 99
	again, _ := s.Get(grid.Name, ndfd, t0)
	if again.Values["temp"][0] != 1 {
		t.Errorf("a reader's change reached the store: read again as %v", again.Values["temp"][0])
	}
}
