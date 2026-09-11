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
	"strconv"
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

// segmentsFromScript is the join read the other way (F-91): a card's SCRIPT
// becomes the segments the broadcast engine plays.
//
// WHY THE ROUND TRIP EXISTS AT ALL. A location report is composed ONCE, at
// standby, and what survives onto the card is Text — lineup.Part has nowhere to
// put a Role, a self-introduction or a pause, and DR-1 keeps the domain out of
// the schedule. So the reader is handed the card's words rather than the
// segments they came from, and this is where they become playable again.
//
// THE CARD IS THE TRUTH, AND THAT IS THE POINT. Recomposing at read time would
// be a second Composer: eleven more requests, and words that could differ from
// the ones the operator has been reading in the slot for the last five minutes.
// A station that says something other than what it showed is the failure
// stale.go's header is about, arrived at from the other side.
//
// THE ROLE IS DELIBERATELY UNSET. synth.Segment's zero Role is cast.All — the
// root correspondent — so the main track reads in the station's own voice.
// The rotation's per-report roles do not survive the card (G-7, recorded at
// p3-flip-design.md and reopened as F-92); what CANNOT differ is the words.
//
// THE KEY NAMES THE CARD. A Source caches rendered PCM by (key, voice), and the
// Source is built per read and discarded with it — but two cards composed for
// the same location would collide on a key made of the text alone, and the
// second would play the first's audio. The card's id is what makes them
// distinct, because it is what the Director guarantees is unique.
func segmentsFromScript(id string, sc lineup.Script) []synth.Segment {
	var segs []synth.Segment
	for i, p := range sc.Parts { // bounded by the script (P10-02)
		// A BLANK PART IS DROPPED, the rule scriptFromSegments states going the
		// other way: a segment with nothing in it renders nothing and the
		// Source treats a failed render as the end of the broadcast.
		if strings.TrimSpace(p.Text) == "" {
			continue
		}
		segs = append(segs, synth.Segment{Key: "card:" + id + ":" + strconv.Itoa(i), Text: p.Text})
	}
	return segs
}
