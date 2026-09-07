package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/severe"
	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/render"
)

// classifier_crosstable_test.go — the four-way cross-table over NWS product
// strings (0.15.0 DISCOVER, brief handoff area 2: "that table IS the
// requirement").
//
// FOUR CLASSIFIERS READ THE SAME STRINGS, and the first draft of the discovery
// report named two. They answer different questions, which is why nobody
// noticed they were the same shape:
//
//	severe.Classify       -> which TAB of the [w] window        (partial: (Tab, bool))
//	globalfeed.LaneOf     -> which LANE of the marquee band     (total: defaults to Warnings)
//	cast.Classify         -> which TONE sounds before the words (total: defaults to Warning)
//	render.AlertIsWarning -> warning-vs-advisory for COLOUR     (total: defaults to false)
//
// THE TABLE IS EMITTED, NOT JUST ASSERTED. `go test -run TestTheProductClassifierCrossTable -v`
// prints it, and the discovery report quotes that output. A table nobody can
// regenerate is a table nobody can trust after the next edit.
func TestTheProductClassifierCrossTable(t *testing.T) {
	type row struct {
		product string
		class   globalfeed.Class
	}
	// THE CURATED QUERY IS DERIVED, NOT COPIED (FR-2.2). A product added to
	// severeEvents() arrives here on its own and moves the divergence count
	// below, so the coupling between the feed's closed set and the four
	// classifiers cannot drift. A hand-copied list is what let this table look
	// complete while severeEvents() moved underneath it.
	var products []row
	for _, p := range globalfeed.CuratedProducts() {
		products = append(products, row{p, globalfeed.ClassSevereWx})
	}
	products = append(products, []row{
		// The rest of the civil-emergency family (globalfeed/civil.go): these
		// reach the window through tracked locations, not the national query.
		{"Civil Emergency Message", globalfeed.ClassSevereWx},
		{"Local Area Emergency", globalfeed.ClassSevereWx},
		{"Child Abduction Emergency", globalfeed.ClassSevereWx},
		{"Blue Alert", globalfeed.ClassSevereWx},
		{"911 Telephone Outage", globalfeed.ClassSevereWx},
		{"Extreme Fire Danger", globalfeed.ClassSevereWx},
		// The arms severe.Classify has and LaneOf does not. THIS IS THE POINT.
		{"Small Craft Advisory", globalfeed.ClassSevereWx},
		{"Flood Advisory", globalfeed.ClassSevereWx},
		{"Air Quality Alert", globalfeed.ClassSevereWx},
		{"Coastal Flood Statement", globalfeed.ClassSevereWx},
		{"Special Weather Statement", globalfeed.ClassSevereWx},
		{"Marine Weather Statement", globalfeed.ClassSevereWx},
		{"Hazardous Weather Outlook", globalfeed.ClassSevereWx},
		// The two classes LaneOf decides before it ever reads the string.
		{"Earthquake", globalfeed.ClassQuake},
		{"Hurricane", globalfeed.ClassTropical},
		// Unknown: every classifier's default, side by side.
		{"Nonexistent Product Type", globalfeed.ClassSevereWx},
	}...)

	label := func(c category.Category) string {
		if c == category.None {
			return "—none—"
		}
		return c.Label()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n%-30s | %-16s | %-16s | %-10s | %s\n",
		"PRODUCT", "severe → tab", "LaneOf → lane", "cast → tone", "AlertIsWarning")
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 100))

	diverged := 0
	for _, r := range products {
		tab, shown := severe.Classify(r.class, r.product)
		lane := globalfeed.LaneOf(globalfeed.Event{Class: r.class, Type: r.product})
		tone := cast.Classify(r.product)
		warn := render.AlertIsWarning(r.product, "")

		tabTxt := label(tab)
		if !shown {
			tabTxt = "NOT SHOWN"
		}
		mark := ""
		// The divergence that matters: the window and the band disagree about
		// the same product. Both answer "which category", so they should agree.
		if shown && tab != lane {
			mark = "   <== WINDOW AND BAND DISAGREE"
			diverged++
		}
		fmt.Fprintf(&b, "%-30s | %-16s | %-16s | %-10s | %-5v%s\n",
			r.product, tabTxt, label(lane), tone, warn, mark)
	}
	fmt.Fprintf(&b, "\n%d of %d products are filed differently by the window and the band.\n",
		diverged, len(products))
	t.Log(b.String())

	// THE ASSERTION, so this is a gate and not a report. The count is pinned:
	// a classifier edit that changes how many products disagree fails here and
	// has to be looked at, whichever direction it moves.
	const wantDiverged = 7
	if diverged != wantDiverged {
		t.Errorf("window/band divergence count is %d, pinned at %d — a classifier changed; "+
			"read the table above and re-ratify before moving the pin", diverged, wantDiverged)
	}
}
