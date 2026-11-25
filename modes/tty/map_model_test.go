package tty

// map_model_test.go — the map model's seam (0.19.6 design MS-1, MS-2): its
// values and their translation, and how a test calls the model.

import (
	"reflect"
	"testing"
)

// withModel is d with its map model replaced by mm, the answer of one of the
// model's methods called with d's Env: the result folded back (0.19.6 design
// MS-7).
func withModel(d Dashboard, mm mapModel) Dashboard {
	d.mapModel = mm
	return d
}

// envOf is d's map model and the Env it is called with, for a method
// expression on what a helper returns: mapModel.mapTitle(envOf(openMap(…))).
func envOf(d Dashboard) (mapModel, mapEnv) { return d.mapModel, d.mapEnv() }

// TestTheMapsWindowsTranslateBothWays holds the one switch each way between
// the Dashboard's windows and the map's (0.19.6 design MS-2, A-31): each of
// the map's is one of the Dashboard's and back, and no other window of the
// Dashboard's is the map's.
func TestTheMapsWindowsTranslateBothWays(t *testing.T) {
	theMaps := 0
	for m := modalNone; m < numModals; m++ { // every window (P10-02)
		w := mapModalOf(m)
		if w == mapModalNone {
			continue
		}
		theMaps++
		if back := w.modal(); back != m {
			t.Errorf("the Dashboard's %s is the map's %d, which is the Dashboard's %s again", modalName(m), w, modalName(back))
		}
	}
	for w := mapModalMap; w <= mapModalPropFreq; w++ { // the map's five (P10-02)
		if mapModalOf(w.modal()) != w {
			t.Errorf("the map's window %d is the Dashboard's %s, which is the map's %d", w, modalName(w.modal()), mapModalOf(w.modal()))
		}
	}
	if mapModalNone.modal() != modalNone {
		t.Errorf("the map's none is the Dashboard's %s", modalName(mapModalNone.modal()))
	}
	if theMaps != 5 {
		t.Fatalf("%d of the Dashboard's windows are the map's; the map has 5 (design MS-2): a window added on one side only", theMaps)
	}
}

// TestTheEnvLendsNoFuncButTheClockAndTheTicks holds 0.19.6 D-21: the map's
// Env carries plain values - no hook, no interface - and no func field but
// the Dashboard's own clock and tick hook, at any depth of its fields.
func TestTheEnvLendsNoFuncButTheClockAndTheTicks(t *testing.T) {
	type at struct {
		path string
		ty   reflect.Type
	}
	queue := []at{{"mapEnv", reflect.TypeOf(mapEnv{})}}
	seen := map[reflect.Type]bool{}
	var funcs []string
	for len(queue) > 0 { // each type once: bounded by the types the Env names (P10-02)
		next := queue[0]
		queue = queue[1:]
		switch next.ty.Kind() {
		case reflect.Func:
			funcs = append(funcs, next.path)
			continue
		case reflect.Interface:
			t.Errorf("%s is an interface: the Env lends plain values (0.19.6 design MS-2)", next.path)
			continue
		}
		if seen[next.ty] {
			continue
		}
		seen[next.ty] = true
		switch next.ty.Kind() {
		case reflect.Struct:
			for i := range next.ty.NumField() { // bounded by the struct (P10-02)
				f := next.ty.Field(i)
				queue = append(queue, at{next.path + "." + f.Name, f.Type})
			}
		case reflect.Slice, reflect.Array, reflect.Pointer:
			queue = append(queue, at{next.path + "[]", next.ty.Elem()})
		case reflect.Map:
			queue = append(queue, at{next.path + "{key}", next.ty.Key()}, at{next.path + "{}", next.ty.Elem()})
		}
	}
	if want := []string{"mapEnv.Now", "mapEnv.Ticks"}; !reflect.DeepEqual(funcs, want) {
		t.Errorf("the Env's func fields are %v; want %v, the Dashboard's clock and tick hook alone (0.19.6 D-21)", funcs, want)
	}
}

// TestTheMapsRowsFitTheSettingsView holds mapSettingsView's Visible, a bit
// for each of the map's rows counted from rowMapsOn: every row of the map's
// three groups lies between rowMapsOn and rowMapLayers, and they fit 64 bits.
func TestTheMapsRowsFitTheSettingsView(t *testing.T) {
	if n := rowMapLayers - rowMapsOn + 1; n > 64 {
		t.Fatalf("the map's rows run to %d, past Visible's 64 bits", n)
	}
	mapRows := 0
	for _, r := range setupTable() { // every row (P10-02)
		if r.group != groupMap && r.group != groupLayers && r.group != groupMapLayers {
			continue
		}
		mapRows++
		if r.id < rowMapsOn || r.id > rowMapLayers {
			t.Errorf("row %d is the map's and outside rowMapsOn..rowMapLayers: the Settings view cannot say whether it shows", r.id)
		}
	}
	if mapRows < 3 {
		t.Fatalf("%d rows in the map's groups: the table is not the one this was written on", mapRows)
	}
}
