package app

// rotation_script.go — the join S0 named: a location report's SEGMENTS become
// a card's SCRIPT (0.16.0 P3).
//
// TWO PATHS COMPOSED INTO DIFFERENT SHAPES. The rotation builds
// []synth.Segment and plays it on the engine; a card carries lineup.Script and
// is read through the narrator arbiter. Everything else about the merge was
// already in place — the arbiter serialises, suspends and resumes, pinned by
// eight tests written before this release — so this translation is the join,
// and it is deliberately the only new idea in P3(a).

import (
	"strings"

	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/lineup"
)

// scriptFromSegments translates a composed location report into a card's
// script.
//
// EVERY SEGMENT IS A LINE. PartLine's own comment already says it is "the
// whole of a location report" — the model anticipated this card before
// anything produced one — so there is no head and no tail to invent here.
//
// AND NO TONE. A tone is a promise of a hazard, and the rotation is the
// programme; sounding one before an ordinary report would teach a listener to
// ignore the one that matters.
//
// A BLANK SEGMENT IS DROPPED. A card takes the air with its words already on
// it, and a part with nothing in it is dead air under a callout the band has
// already promised (DR-18). Dropping it here means an all-blank compose
// yields an EMPTY script the executor declines, rather than a card that
// reaches the air with nothing to say.
func scriptFromSegments(segs []synth.Segment) lineup.Script {
	var parts []lineup.Part
	for _, seg := range segs { // bounded by the compose (P10-02)
		if strings.TrimSpace(seg.Text) == "" {
			continue
		}
		parts = append(parts, lineup.Part{Kind: lineup.PartLine, Text: seg.Text})
	}
	if len(parts) == 0 {
		return lineup.Script{}
	}
	return lineup.Script{Parts: parts}
}
