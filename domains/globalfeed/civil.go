package globalfeed

import "github.com/branden-thompson/watchpost/platform/category"

// civil.go — the civil-emergency family: the products that are an INSTRUCTION
// rather than a description.
//
// ONE OWNER (D-1, and F-2's fourth classifier). This table used to live inside
// domains/severe.Classify, where only the [w] window could see it. The producer
// asks the same question when it hands an arrival to the Director, and
// domains/severe imports this package rather than the other way round, so the
// window's copy was the only one and the marquee's lane had no arm for the
// family at all. An evacuation order was therefore laned as an ordinary warning
// — one rung below the rung the ladder reserves for it — and, because the
// national query never asked for the product, it never arrived to be laned
// wrongly in the first place (red team 2026-09-05, C-2).
//
// MATCHED BY NAME, EXACTLY. None of these products names a warning, a watch or
// an advisory, so a keyword classifier cannot reach them; and matching loosely
// on "Emergency" would sweep in products from other programmes on a
// coincidence, which is the trap MVS-D-58 already recorded for "Alert".
type civilProduct struct {
	product  string
	category category.Category
}

// civilEmergencyProducts is the family, with the category each belongs to. A
// function, not a global, per the codebase's table convention (P10-06).
func civilEmergencyProducts() []civilProduct {
	return []civilProduct{
		// EMERGENCY ORDERS (MVS-D-61): the Weather Service's highest-urgency
		// product — leave, now — and the only one that is an instruction rather
		// than a description. Its own category, its own lane, and ReadRank 1.
		{"Evacuation Immediate", category.Emergency},
		{"Civil Emergency Message", category.Disasters}, // hazards that are not weather
		{"Local Area Emergency", category.Disasters},
		{"Child Abduction Emergency", category.Statements},
		{"Blue Alert", category.Statements},
		{"911 Telephone Outage", category.Statements},
		{"Extreme Fire Danger", category.Watches},
	}
}

// CivilEmergencyCategory is the category of a civil-emergency product, and ok
// is false for anything that is not one of them.
func CivilEmergencyCategory(product string) (category.Category, bool) {
	for _, p := range civilEmergencyProducts() { // bounded by the table (P10-02)
		if p.product == product {
			return p.category, true
		}
	}
	return category.Forecasts, false
}
