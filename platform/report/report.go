// Package report is what a location report can be MADE OF, and the only place a
// report kind is described.
//
// THE LIST WILL GROW, AND THAT IS THE DESIGN CONSTRAINT (HUM LEAD, 2026-09-14):
// "while we have a set group of reports now, we WILL have more report types in
// the future, and ensuring Broadcaster is flexible enough for that list to grow
// and change WITHOUT having to completely re-architect the code and flow every
// time is critical."
//
// SO THE TEST OF THIS PACKAGE IS NOT THAT IT DESCRIBES FOUR KINDS. It is what a
// FIFTH costs — and inside this package the answer is one row in `all()`.
// Nothing else here: not the card it travels on, not the event that requests it,
// not the naming rule, not the modal, not the running order.
// `TestAFifthKindCostsOneRow` is that claim made checkable rather than promised.
//
// IT COSTS ONE THING OUTSIDE THIS PACKAGE (F-111). `radioDeck.segments`
// answers each kind with its
// OWN typed hook, so a fifth needs a branch there or the report is requested,
// built, and silently missing what was asked for. The branches are not a defect
// — they cannot be table-driven without erasing the types — but the absence of
// anything NOTICING a fifth was, and `TestEveryReportKindReachesTheComposer`
// (in `app`) is what notices now. A contributor who read only the paragraph
// above would add the row, watch this package's tests go green, and ship a
// silently empty report kind.
//
// Modelled on `platform/category`, whose registry states the same rule: "A
// function rather than a package variable (P10-06), and the ONLY place a
// category is described."
package report

import "strings"

// Kind is one source a location report can carry.
//
// THE ZERO VALUE IS A REAL KIND, deliberately: the NWS forecast is the report
// nearly every request wants, and making it the zero saves nothing and would
// cost a "none" member that `Set` already expresses by being empty.
type Kind uint8

const (
	NWS Kind = iota
	Marine
	Fire
	Seismic

	// numKinds is the sentinel, and it is what `All` counts to. A new kind is
	// added ABOVE this line and below the last real one.
	numKinds
)

// Spec is how a kind presents itself. Both forms are here because both are used
// and they are used in different places (HUM LEAD, 2026-09-14): "We show both in
// the modal, only the Label in the Scheduled-Line up table."
type Spec struct {
	// FullName is what the modal calls it — the operator is CHOOSING, and a
	// four-letter label is not enough to choose from.
	FullName string

	// Label is what the running order calls it, where a row has ~20 cells for
	// the whole set and the operator is READING BACK a decision already made.
	Label string
}

// all is the registry. A function rather than a package variable (P10-06), and
// the ONLY place a kind is described.
//
// ORDER IS THE REGISTRY'S, NOT THE OPERATOR'S. `Describe` lists labels in this
// order whatever order they were chosen in, so one set of kinds is always one
// string — two cards carrying the same reports read the same, and the running
// order does not shuffle when a request is re-made.
func all() [numKinds]Spec {
	return [numKinds]Spec{
		NWS:     {FullName: "NWS Forecast", Label: "NWS"},
		Marine:  {FullName: "NDBC Marine Forecast", Label: "MARINE"},
		Fire:    {FullName: "NIFC & NASA FIRMS Fire & Hotspot", Label: "FIRE"},
		Seismic: {FullName: "USGS Seismic", Label: "QUAKE"},
	}
}

// Of describes a kind. An unknown kind returns the zero Spec rather than
// panicking: a report set read from a file written by a later version names a
// kind this build does not have, and the honest answer is to say nothing about
// it rather than to stop.
func Of(k Kind) Spec {
	if k >= numKinds {
		return Spec{}
	}
	return all()[k]
}

// All is every kind, in registry order.
func All() []Kind {
	out := make([]Kind, 0, numKinds)
	for k := Kind(0); k < numKinds; k++ { // bounded by the registry (P10-02)
		out = append(out, k)
	}
	return out
}

// Set is which kinds a report carries.
//
// A BITSET RATHER THAN A SLICE, and the reasons are all about the seams it
// crosses. It is COMPARABLE — so it works in a memo key, in `==`, and in a
// golden — where a slice would have needed an equality helper at every one of
// them. Its zero value means "nothing chosen", which is the state a half-filled
// modal is in. And it costs the same at twenty kinds as at four.
type Set uint32

// Add returns the set with k in it.
func (s Set) Add(k Kind) Set {
	if k >= numKinds {
		return s
	}
	return s | 1<<k
}

// Without returns the set with k out of it.
func (s Set) Without(k Kind) Set { return s &^ (1 << k) }

// Toggle flips one kind, which is what a modal row does.
func (s Set) Toggle(k Kind) Set {
	if s.Has(k) {
		return s.Without(k)
	}
	return s.Add(k)
}

// Has reports whether the set carries k.
func (s Set) Has(k Kind) bool { return k < numKinds && s&(1<<k) != 0 }

// Empty reports whether nothing is chosen — the state a request cannot be made
// in, and the reason the modal's Schedule is refused until something is.
func (s Set) Empty() bool { return s.Kinds() == nil }

// Full reports whether every kind the build knows is chosen.
func (s Set) Full() bool { return len(s.Kinds()) == int(numKinds) }

// Kinds is the chosen kinds in REGISTRY ORDER.
func (s Set) Kinds() []Kind {
	var out []Kind
	for k := Kind(0); k < numKinds; k++ { // bounded by the registry (P10-02)
		if s.Has(k) {
			out = append(out, k)
		}
	}
	return out
}

// Everything is the set with every kind in it.
func Everything() Set {
	var s Set
	for _, k := range All() { // bounded by the registry (P10-02)
		s = s.Add(k)
	}
	return s
}

// FullLabel is what a complete report calls itself in the running order.
//
// THE MOCK'S OWN WORDS. A card carrying everything says what it is rather than
// listing four labels the operator would have to add up.
const FullLabel = "Location Report, Full"

// Describe is what the running order draws for a set (HUM LEAD, 2026-09-14):
// everything reads "Location Report, Full"; anything else is the LABELS,
// comma-delimited, in registry order.
//
// ONE RULE FOR ONE, TWO OR THREE. The first spec said a single choice shows its
// NAME and several show labels — the ruling simplified that to labels
// throughout, and a single kind is simply a one-element list. A special case for
// one would be a second naming rule to keep in step with the first.
//
// AN EMPTY SET DESCRIBES ITSELF AS NOTHING, and the caller decides what to draw:
// there is no card in that state, and inventing a word for it here would put a
// sentence on a screen for a state the schedule cannot hold.
func (s Set) Describe() string {
	if s.Empty() {
		return ""
	}
	if s.Full() {
		return FullLabel
	}
	out := make([]string, 0, numKinds)
	for _, k := range s.Kinds() { // bounded by the registry (P10-02)
		out = append(out, Of(k).Label)
	}
	return strings.Join(out, ", ")
}
