package tty

// map_model.go — the map's model (0.19.6 design MS-1): the state the map
// window owns, embedded in the Dashboard. It holds the hooks alone until the
// map's fields and methods join it, together, in B4's third commit (0.19.6
// D-76, D-78).

// mapModel is the map's own state.
type mapModel struct {
	// mapHooks is the map's hooks, held once (0.19.6 A-31, D-77): copies of
	// the Dashboard share it, so a hook is changed only by replacing the
	// pointer, never written through it.
	mapHooks *Hooks
}
