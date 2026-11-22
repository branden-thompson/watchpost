package tty

// ttytest_field_test.go — Field's own test (0.19.6 D-62): it writes through
// unsafe, so a name it cannot honour must stop it rather than write elsewhere.
// internal/ttytest's perturb_test.go once B2 moves it.

import "testing"

// TestFieldWritesItsFieldAndRefusesWhatItCannotFind holds Field to the one
// field it names, and to a panic for a name the type lacks at its top level or
// a field of another type.
func TestFieldWritesItsFieldAndRefusesWhatItCannotFind(t *testing.T) {
	type inner struct{ deep bool }
	type model struct {
		inner
		a, b bool
		n    int
	}
	m := model{inner: inner{deep: true}, a: true, b: true}
	Field[model, bool]("b")(&m, false)
	if !m.deep || !m.a || m.b || m.n != 0 {
		t.Fatalf("Field wrote %+v; only b should have changed", m)
	}
	for _, c := range []struct {
		what string
		get  func()
	}{
		{"a name the type lacks", func() { Field[model, bool]("c") }},
		{"a field of another type", func() { Field[model, bool]("n") }},
		{"a promoted field", func() { Field[model, bool]("deep") }},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Field took %s", c.what)
				}
			}()
			c.get()
		}()
	}
}
