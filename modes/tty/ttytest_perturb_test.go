package tty

// ttytest_perturb_test.go — the memo guards' engine: every field of a surface
// model perturbed by reflection, for any model (0.19.6 D-15). Moved unchanged
// from memo_completeness_test.go, PerturbEach renamed only; internal/ttytest's
// perturb.go once B2 moves it.

import (
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"
)

// PerturbEach is the engine, GENERIC OVER THE MODEL. Two surfaces memoise,
// and the guard is the thing that makes a memo safe to have — so a second copy
// of this walk for the console would be the one duplication that matters: the
// copy would drift, and the surface whose walk stopped descending would report
// coverage it did not have while its frames froze.
//
// `excuse` names the fields it does NOT descend into, and `writer` models a
// writer the app pairs with a field (setup's generation). Both belong to the
// model, not to the walk.
func PerturbEach[T any](t *testing.T, base T, excuse map[string]string, writer func(string, *T), emit func(string, T)) {
	t.Helper()
	walk(t, reflect.TypeOf(base), "", excuse, writer, func(path string, set func(*T)) {
		next := base
		if !apply(&next, set) {
			return // the perturbation panicked the copy; not a finding about the key
		}
		emit(path, next)
	})
}

// apply runs one perturbation, containing a panic from an index a fixture does
// not support. A field that cannot be perturbed safely is skipped rather than
// reported: this test is about the KEY, not about render robustness.
func apply[T any](d *T, set func(*T)) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	set(d)
	return true
}

// walk visits every field this test knows how to change, one level into nested
// structs (relayFault, debug and setup all hold their state that way).
func walk[T any](t *testing.T, typ reflect.Type, prefix string, excuse map[string]string, writer func(string, *T), emit func(string, func(*T))) {
	t.Helper()
	walkAt(t, typ, nil, prefix, excuse, writer, emit)
}

// walkAt is walk at an index path into the model. AN EMBEDDED STRUCT IS
// WALKED AS THE MODEL ITSELF (0.19.5 D-14): its fields are promoted, read as
// the model's own (d.mapDesc), and keep their own names here, so a writer or an
// excuse keyed on a field's name still finds it after the field moved into a
// group.
func walkAt[T any](t *testing.T, typ reflect.Type, at []int, prefix string, excuse map[string]string, writer func(string, *T), emit func(string, func(*T))) {
	t.Helper()
	for i := range typ.NumField() {
		f := typ.Field(i)
		path := prefix + f.Name
		index := append(append([]int(nil), at...), i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			walkAt(t, f.Type, index, prefix, excuse, writer, emit)
			continue
		}
		switch f.Type.Kind() {
		case reflect.Bool, reflect.Int, reflect.Int64, reflect.Float64, reflect.String, reflect.Pointer:
			emit(path, func(d *T) {
				bump(fieldAtPath(d, index))
				if writer != nil {
					writer(path, d) // a field with a paired writer (dashboardWriter)
				}
			})
		case reflect.Struct:
			// ONE LEVEL DOWN, INTO EVERY STRUCT THIS FILE HAS NOT EXCUSED
			// (FR-3.3). A named set — relayFault, debug and setup — would leave a
			// FOURTH window's state unperturbed while the guard reports coverage
			// it does not have. A hand-written list of the windows with state is
			// the same shape as the hand-written memo key it checks, and would
			// miss a new window in exactly the same way.
			if prefix == "" && excuse[path] == "" {
				walkNested(t, f, index, excuse, writer, emit)
			}
		}
	}
}

// walkNested is walk for the fields of one nested struct.
func walkNested[T any](t *testing.T, f reflect.StructField, outer []int, excuse map[string]string, writer func(string, *T), emit func(string, func(*T))) {
	t.Helper()
	for j := range f.Type.NumField() {
		inner := f.Type.Field(j)
		switch inner.Type.Kind() {
		case reflect.Bool, reflect.Int, reflect.Int64, reflect.Float64, reflect.String:
			jdx := j
			name := f.Name
			if excuse[name+"."+inner.Name] != "" {
				continue
			}
			emit(name+"."+inner.Name, func(d *T) {
				v := reflect.ValueOf(d).Elem().FieldByIndex(append(append([]int(nil), outer...), jdx))
				bump(reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem())
				if writer != nil {
					writer(name, d)
				}
			})
		}
	}
}

// fieldAtPath is one addressable field of d, by its index path: the top level,
// or a field of an embedded struct.
func fieldAtPath[T any](d *T, index []int) reflect.Value {
	v := reflect.ValueOf(d).Elem().FieldByIndex(index)
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

// bump changes a value to a DIFFERENT one of its own type. The particular value
// does not matter; only that it differs.
func bump(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(!v.Bool())
	case reflect.String:
		v.SetString(v.String() + "·changed")
	case reflect.Int, reflect.Int64:
		if v.Type() == reflect.TypeOf(time.Duration(0)) {
			v.SetInt(v.Int() + int64(time.Second))
			return
		}
		v.SetInt(v.Int() + 1)
	// FLOATS ARE PERTURBED. A walk that lists the kinds it handles and falls
	// through the rest in silence is the same shape as the hand-written key it
	// exists to check, and leaves a float input to a memoised frame (the fire
	// threshold, a radius) outside the guard.
	case reflect.Float64:
		v.SetFloat(v.Float() + 1)
	// AND SO ARE POINTERS (D-129). A *ref the frame READS THROUGH — the search
	// window's pooled match — is otherwise no emitted kind at all, and the
	// guard reports the window covered while the field behind half its
	// sentences is never perturbed.
	//
	// A COPY, NEVER THE SHARED POINTEE. PerturbEach copies the MODEL, and a
	// copy of a struct shares every pointer in it — so bumping through one
	// would move the BASE as well, `before` and `after` would agree, and the
	// guard would pass by changing nothing. Cloning is what makes including
	// pointers safe: a fresh pointee leaves the baseline untouched, and the new
	// POINTER is itself a difference any key holding it will see.
	//
	// NIL IS LEFT ALONE. Production gives these windows nil; minting a value
	// there would perturb into a state the app never has.
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		fresh := reflect.New(v.Type().Elem())
		fresh.Elem().Set(v.Elem())
		if fresh.Elem().Kind() == reflect.Struct {
			if f := fresh.Elem().Field(0); f.CanSet() {
				bump(f)
			}
		} else {
			bump(fresh.Elem())
		}
		v.Set(fresh)
	}
}

// Field is a writer for one unexported field of T, found by name once: the
// engine's own kind of write, for a test that must put a value back in a state
// no exported name sets (0.19.6 D-62). Resolved at a test file's package
// scope, the write itself allocates nothing inside an allocation pin. A name T
// does not have at its top level, or a field of another type than V, panics,
// so the test binary stops before any test runs rather than writing elsewhere.
func Field[T, V any](name string) func(*T, V) {
	f, ok := reflect.TypeFor[T]().FieldByName(name)
	if !ok || len(f.Index) != 1 || f.Type != reflect.TypeFor[V]() {
		panic(fmt.Sprintf("Field: %s has no field %s of type %s", reflect.TypeFor[T](), name, reflect.TypeFor[V]()))
	}
	return func(p *T, v V) { *(*V)(unsafe.Add(unsafe.Pointer(p), f.Offset)) = v }
}
