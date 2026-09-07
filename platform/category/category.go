// Package category is THE registry of alert categories.
//
// One concept was declared four times — the severe window's tab, the window
// layer's mirror of it, the ticker's lane, and the national stack's capping
// bucket — each with a list of its own kept beside it. Adding a category meant
// editing six places, and forgetting any one of them failed silently: a lane
// missing from the rotation never reached the band, a bucket missing from the
// cap dropped its alerts from the stack, and two enums disagreeing filed every
// row one tab over. All three happened in 0.14.0 (F-21).
//
// So the categories are declared once, here, and everything else is a view of
// this. It lives under platform/ because scripts/lint-imports.sh forbids
// anything in modes/ from importing domains/*, and both layers need it —
// platform/snapshot is the precedent.
package category

import (
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
)

// Category is one row of the registry. The order is the order the window's tabs
// read in, most serious first: an evacuation order leads even when it is empty,
// because a category you have to go looking for is the wrong shape for it.
type Category int

const (
	Emergency Category = iota
	Warnings
	Watches
	Advisories
	Statements
	Disasters
	Marine
	Forecasts
	Count          // the number of categories; never a valid one
	None  Category = -1
)

// Spec is everything the app needs to know about a category.
//
// A field left at its zero value MEANS something here: no Band is no ticker
// lane, and Rotation is only read when there is one.
type Spec struct {
	// Bucket is the full name — "Emergency Orders". TabLabel is what a tab
	// wears when there is room, TabShort when there is not.
	Bucket, TabLabel, TabShort string

	// Tint is the window's row tint. BandTone is the ticker's band colour, and
	// is EMPTY for a category with no lane. BandLabel is what that band calls
	// itself, which can differ from the tab's word because the band has a whole
	// strip to say it in.
	Tint, BandTone render.Token
	BandLabel      string

	// Rotation is where the lane sits in the marquee's cycle. Ignored when
	// Band is empty.
	Rotation int

	// ReadRank is where the category sits in a spoken burst — 1 leads, and
	// ZERO MEANS NEVER READ AS AN ALERT (Forecasts: an outlook is what might
	// happen, and the burst is for what is).
	//
	// A THIRD ORDERING, and deliberately not the other two. The tab bar reads
	// down by seriousness, the marquee leads with the urgent and the lanes the
	// national feed fills, and this decides what a listener hears first. Their
	// disagreement is the design: a disaster is read before a warning (R-2)
	// while the tab bar places it lower. TestTheThreeOrderingsAreIndependently
	// Declared fails loudly if anyone tidies them into one, because doing that
	// silently changes what is spoken.
	ReadRank int

	// Watchlist marks a category the national feed cannot produce, whose rows
	// arrive only through tracked locations — so an empty tab can say why.
	Watchlist bool
}

// all is the registry, indexed by Category. A function rather than a package
// variable (P10-06), and the ONLY place a category is described.
func all() [Count]Spec {
	return [Count]Spec{
		Emergency: {Bucket: "Emergency Orders", TabLabel: "Emergency", TabShort: "Emrg",
			Tint: render.EventCatEmergencyBG, BandTone: render.TickerEmergencyBG, BandLabel: "Emergency Orders", Rotation: 0, ReadRank: 1},
		Warnings: {Bucket: "Warnings", TabLabel: "Warnings", TabShort: "Warn",
			Tint: render.EventCatWarningBG, BandTone: render.TickerWarningBG, BandLabel: "Warnings", Rotation: 3, ReadRank: 3},
		Watches: {Bucket: "Watches", TabLabel: "Watches", TabShort: "Watch",
			Tint: render.EventCatWatchBG, BandTone: render.TickerWatchBG, BandLabel: "Watches", Rotation: 4, ReadRank: 4},
		Advisories: {Bucket: "Advisories", TabLabel: "Advisories", TabShort: "Advis",
			Tint: render.EventCatAdvisoryBG, BandTone: render.TickerAdvisoryBG, BandLabel: "Advisories", Rotation: 5, Watchlist: true, ReadRank: 5},
		Statements: {Bucket: "Special Statements", TabLabel: "Spec. Statements", TabShort: "Stmts",
			Tint: render.EventCatStmtBG, BandTone: render.TickerStatementBG, BandLabel: "Statements", Rotation: 6, Watchlist: true, ReadRank: 6},
		Disasters: {Bucket: "Disasters", TabLabel: "Disasters", TabShort: "Disast",
			Tint: render.EventCatDisasterBG, BandTone: render.TickerDisasterBG, BandLabel: "Disasters", Rotation: 1, ReadRank: 2},
		Marine: {Bucket: "Marine", TabLabel: "Marine", TabShort: "Marine",
			Tint: render.EventCatMarineBG, BandTone: render.TickerMarineBG, BandLabel: "Marine", Rotation: 2, ReadRank: 7},
		// FORECASTS HAS NO LANE. The marquee is for what is happening; an
		// outlook is what might happen, days out, over a whole forecast area
		// (MVS-D-59).
		Forecasts: {Bucket: "Forecasts and Outlooks", TabLabel: "Forecasts", TabShort: "Fcast",
			Tint: render.EventCatForecastBG, Watchlist: true},
	}
}

// Of is one category's spec. A category outside the registry returns the zero
// Spec rather than panicking: this is the alert path, and a wrong label is
// better than no frame.
func Of(c Category) Spec {
	if c < 0 || c >= Count {
		return Spec{}
	}
	s := all()[c]
	// A HOLE IN THE TABLE IS CAUGHT WHERE IT IS USED, not only by the package's
	// own test: everything in the app reads categories through here, so an
	// entry added to the enum without a row beside it would otherwise render as
	// a blank tab with no colour and no complaint.
	if err := invariant.Check(s.TabLabel != "", "every category in the enum has a row in the registry"); err != nil {
		return Spec{}
	}
	return s
}

// All is every category in tab order.
func All() []Category {
	if err := invariant.Check(Count > 0, "the registry describes at least one category"); err != nil {
		return nil
	}
	out := make([]Category, 0, Count)
	for c := Category(0); c < Count; c++ {
		out = append(out, c)
	}
	if err := invariant.Check(len(out) == int(Count), "All lists every category exactly once"); err != nil {
		return nil
	}
	return out
}

// Lanes is every category that has a ticker lane, in ROTATION order — which is
// not tab order: the band leads with the most urgent and then the two the
// national feed fills, while the tabs read straight down by seriousness.
func Lanes() []Category {
	byRotation := map[int]Category{}
	for _, c := range All() {
		if s := Of(c); s.BandTone != "" {
			byRotation[s.Rotation] = c
		}
	}
	if err := invariant.Check(len(byRotation) <= int(Count), "no more lanes than there are categories"); err != nil {
		return nil
	}
	out := make([]Category, 0, len(byRotation))
	for i := 0; i < len(byRotation); i++ {
		// THE ROTATION MUST BE DENSE. This walks 0..n-1, so a gap in the
		// numbering would end the walk early and drop every lane after it from
		// the band — silently, which is the failure this package exists to make
		// impossible.
		if err := invariant.Check(byRotation[i] != 0 || i == 0, "the lane rotation is numbered without gaps"); err != nil {
			return out
		}
		out = append(out, byRotation[i])
	}
	return out
}

// ReadOrder is every category a spoken burst may reach, in READ order — the
// third ordering, and neither of the other two: Emergency Orders lead, and a
// disaster is read before a warning (R-2) although the tab bar places it lower.
// A category with no rank is never read as an alert.
func ReadOrder() []Category {
	byRank := map[int]Category{}
	for _, c := range All() {
		if s := Of(c); s.ReadRank > 0 {
			byRank[s.ReadRank] = c
		}
	}
	out := make([]Category, 0, len(byRank))
	for i := 1; i <= len(byRank); i++ {
		// THE RANKS MUST BE DENSE FROM 1. A gap or a repeat would drop a
		// category out of every burst — a listener would simply stop hearing
		// marine alerts, with nothing anywhere to say why. Bounded by the map
		// (P10-02).
		//
		// The two-value lookup rather than a zero test: Emergency is Category(0)
		// AND rank 1, so comparing the value against 0 would need a special case
		// to survive, and a special case in a density check is how the gap gets
		// back in.
		c, ok := byRank[i]
		if err := invariant.Check(ok, "the read ranks are numbered from 1 without gaps"); err != nil {
			return out
		}
		out = append(out, c)
	}
	return out
}

// HasLane reports whether this category reaches the marquee at all.
func HasLane(c Category) bool {
	s := Of(c)
	// A HALF-SET LANE IS THE DANGEROUS STATE: a colour with no word leaves a
	// band only a sighted reader with colour vision can identify (R-12a), and a
	// word with no colour leaves it unreadable against the frame. Either way
	// this must not report a usable lane.
	if err := invariant.Check(s.BandTone == "" || s.BandLabel != "", "a lane with a colour also has a name"); err != nil {
		return false
	}
	return s.BandTone != ""
}

// Label is what the ticker band calls this lane — the marquee's own word, which
// is not always the tab's: the band has a whole strip to say it in.
//
// THE BAND SAYS ITS LANE IN WORDS. Lanes rotate through one strip and the only
// other thing distinguishing them is the background colour, which a colourblind
// reader, a monochrome theme and a NO_COLOR terminal all flatten to nothing
// (R-12a).
func (c Category) Label() string {
	s := Of(c)
	if err := invariant.Check(s.BandLabel == "" || s.BandTone != "", "a lane that names itself also has a colour"); err != nil {
		return ""
	}
	return s.BandLabel
}
