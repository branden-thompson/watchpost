package history

import (
	"bytes"
	"errors"
	"math"
	"strconv"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// maxDecimals bounds a field's kept decimals: past it, rounding is noise.
const maxDecimals = 9

// values is a field's values as held and kept: a missing one NaN in memory and
// null on disk, one allocation a field, not one a value (FR-6.5, PF-F5).
type values []float64

// MarshalJSON writes values as a JSON array, a missing one null, each number
// as encoding/json writes a float64.
func (v values) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	out := make([]byte, 0, 2+8*len(v))
	out = append(out, '[')
	for i, x := range v { // bounded by the shape (P10-02)
		if i > 0 {
			out = append(out, ',')
		}
		if math.IsNaN(x) || math.IsInf(x, 0) {
			out = append(out, "null"...)
			continue
		}
		out = appendNumber(out, x)
	}
	return append(out, ']'), nil
}

// appendNumber is x appended as encoding/json writes a float64: exponent form
// only below 1e-6 or from 1e21, and a two-digit negative exponent's 0 dropped.
func appendNumber(b []byte, x float64) []byte {
	format := byte('f')
	if abs := math.Abs(x); abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		format = 'e'
	}
	b = strconv.AppendFloat(b, x, format, -1, 64)
	if n := len(b); format == 'e' && n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
		b[n-2] = b[n-1]
		b = b[:n-1]
	}
	return b
}

// errNotValues is a document's values that are not numbers and nulls.
var errNotValues = errors.New("history: values are numbers and nulls")

// UnmarshalJSON reads a JSON array of numbers and nulls, a null as NaN; null
// itself is no values. encoding/json has checked the syntax before it calls
// this, so an element is whole between commas unless it is a string, an
// object or an array, each of which is refused.
func (v *values) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if string(raw) == "null" {
		*v = nil
		return nil
	}
	if len(raw) < 2 || raw[0] != '[' || raw[len(raw)-1] != ']' {
		return errNotValues
	}
	body := bytes.TrimSpace(raw[1 : len(raw)-1])
	if len(body) == 0 {
		*v = values{}
		return nil
	}
	n := bytes.Count(body, []byte{','}) + 1
	out := make(values, 0, n)
	for range n { // one element a pass (P10-02)
		elem, rest, _ := bytes.Cut(body, []byte{','})
		elem, body = bytes.TrimSpace(elem), rest
		if string(elem) == "null" {
			out = append(out, math.NaN())
			continue
		}
		if len(elem) == 0 || !numberBytes(elem) {
			return errNotValues
		}
		x, err := strconv.ParseFloat(string(elem), 64)
		if err != nil {
			return errNotValues // past a float64's range
		}
		out = append(out, x)
	}
	*v = out
	return nil
}

// numberBytes reports whether every byte of b is one a JSON number holds.
func numberBytes(b []byte) bool {
	for _, c := range b { // bounded by the element (P10-02)
		if !numberByte[c] {
			return false
		}
	}
	return true
}

// numberByte is, by byte, whether a JSON number holds it.
var numberByte = func() (t [256]bool) {
	for _, c := range []byte("0123456789+-.eE") {
		t[c] = true
	}
	return t
}()

// encode is values as kept: each rounded to decimals, a missing one NaN.
func encode(v []float64, decimals int) values {
	if len(v) == 0 {
		return nil
	}
	if err := invariant.Check(decimals >= 0 && decimals <= maxDecimals, "a field's decimals are 0 to 9"); err != nil {
		return nil
	}
	scale := math.Pow(10, float64(decimals))
	out := make(values, len(v))
	for i, x := range v { // bounded by the shape (P10-02)
		out[i] = math.NaN() // missing: null on disk
		if !math.IsNaN(x) && !math.IsInf(x, 0) {
			out[i] = math.Round(x*scale) / scale
		}
	}
	return out
}

// decode is values as a reader is given them: its own copy, so nothing it
// does reaches the days the store holds.
func decode(v values) []float64 {
	if len(v) == 0 {
		return nil
	}
	return append([]float64(nil), v...)
}
