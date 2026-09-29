package tty

// window_keys.go — which keys the window shown owns (0.18.0 F-184, D-107):
// one declaration a window, read by the key routing, by Help, and by the
// guards that hold every window to it.
//
// THE ROUTING WAS IN THREE PLACES. handleKey's switch named the windows that
// take every key and the one that takes the keys it binds; handleNav's
// hand-written list named the windows whose scroll the arrows walk - and a
// window missing from it drew a scroll rail whose arrows reached PAST it to
// the table underneath; Help named the map's group by its title. Now each
// window says it once, here, and a window without a declaration fails its
// guard the moment it is added (window_keys_test.go).
//
// THE STACK ORDERS THEM (D-107): only the window shown - the top of the stack -
// is asked; the windows under it never see a key. What it does not own goes
// on to the Observer's own bindings. The Router, above both, keeps the
// surface's keys - the swap, and the console's windows - and is outside this.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/term"
)

// keyClaim is how much of the keyboard a window takes while it is shown.
type keyClaim uint8

const (
	// claimActions is none of its own: the Observer's bindings reach it, and
	// its nav, if it scrolls or walks, is the window's.
	claimActions keyClaim = iota
	// claimBound is the keys it binds (the map, D-61): the rest go on to the
	// Observer's bindings.
	claimBound
	// claimAll is every key: a form or a search, which a stray key must never
	// leave (Setup, Add, Remove, Request).
	claimAll
)

// windowKeys is one window's declaration.
type windowKeys struct {
	claim keyClaim
	// all takes every key (claimAll).
	all func(Dashboard, tea.KeyPressMsg) (tea.Model, tea.Cmd)
	// bound are the keys it binds, and onKey takes one of them (claimBound);
	// onKey says false for a key it does not own.
	bound func(Dashboard) term.KeyMap
	onKey func(Dashboard, tea.KeyPressMsg) (tea.Model, tea.Cmd, bool)
	// nav is what the Observer's selection and paging actions do while it is
	// shown: its scroll, its rows, its fields. Nil walks the table - which is
	// only right for a window that leaves the arrows to it on purpose.
	nav func(Dashboard, term.Action) Dashboard
	// helpTitle, helpActions and helpRows are its group in Help, when its keys
	// are its own (claimBound): the title, the actions, a row of each idea.
	helpTitle   string
	helpActions []term.Action
	helpRows    func(keys term.KeyMap, ascii bool) []mapHelpRow
}

// windowKeysOf is a window's declaration, and false for one without - which
// window_keys_test.go refuses for every window there is.
func windowKeysOf(m modal) (windowKeys, bool) {
	// A METHOD EXPRESSION, NOT A CLOSURE (W14, S-1): P10's call graph reads a
	// closure's body as a call made here, and that one false edge closed a
	// 32-function "recursion" through the layout code; nothing here calls it.
	scrolls := Dashboard.handleModalNav
	switch m {
	case modalNone:
		return windowKeys{claim: claimActions}, true // the dashboard: its bindings, its table
	case modalHelp, modalDetails, modalAlerts, modalStatus, modalAbout, modalCard:
		return windowKeys{claim: claimActions, nav: scrolls}, true
	case modalSevere:
		return windowKeys{claim: claimActions, nav: Dashboard.handleSevereNav}, true // 0.13.0: tabs and rows, or the record's scroll
	case modalRelayFault:
		return windowKeys{claim: claimActions, nav: Dashboard.handleRelayFaultNav}, true // MVS-D-76: the three ways out of a dead relay
	case modalDebug:
		return windowKeys{claim: claimActions, nav: Dashboard.handleDebugNav}, true // F-21
	case modalSetup:
		return windowKeys{claim: claimAll, all: Dashboard.handleSetupKey}, true
	case modalAdd:
		return windowKeys{claim: claimAll, all: Dashboard.handleAddKey}, true // its own keys walk `selected`
	case modalRemove:
		return windowKeys{claim: claimAll, all: Dashboard.handleRemoveKey}, true
	case modalRequest:
		return windowKeys{claim: claimAll, all: Dashboard.handleRequestKey, nav: Dashboard.handleRequestNav}, true // R4: a form, walked field by field
	case modalMap:
		return windowKeys{claim: claimBound, bound: func(d Dashboard) term.KeyMap { return d.mapKeys }, onKey: Dashboard.handleMapKey,
			helpTitle: "MAP", helpActions: mapActions, helpRows: mapHelpRows}, true // D-61: the open map owns the keys it binds
	}
	return windowKeys{}, false
}

// routeWindowKey offers a key to the window shown (F-184): all of them to a
// window that claims every key, its own to a window that binds some. False
// when the key goes on to the Observer's bindings.
func (d Dashboard) routeWindowKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	w, _ := windowKeysOf(d.modal)
	switch w.claim {
	case claimAll:
		m, cmd := w.all(d, key)
		return m, cmd, true
	case claimBound:
		return w.onKey(d, key)
	}
	return d, nil, false
}

// windowHelpGroups are the windows' own groups in Help, from their
// declarations, in the windows' order.
func windowHelpGroups() []helpGroup {
	var out []helpGroup
	for m := modalNone; m < numModals; m++ {
		if w, ok := windowKeysOf(m); ok && w.helpTitle != "" {
			out = append(out, helpGroup{name: w.helpTitle, actions: w.helpActions, rows: w.helpRows})
		}
	}
	return out
}
