import pathlib
# MVS-D-77 undone: the schedule goes back to one card per ALERT, so a burst of
# three is three entries the operator must promote or drop one at a time — and
# DR-7's one-ahead silently changes scale, building the second ALERT of a burst
# rather than the card behind the takeover.
#
# RE-ANCHORED at the T3.10 red team: Burst.Cards is gone, so the defect is now
# expressed by splitting the takeover back into one card per ref.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """	if b.HasTakeover() {
		if next, err := d.lineup.Queue(AlertRail, b.Takeover); err == nil {
			d.lineup = next // an error is "already held": a repeated burst is not a second read
		}
	}"""
new = """	for _, ref := range b.Takeover.Refs {
		c := b.Takeover
		c.ID, c.Refs = ref, []string{ref}
		if next, err := d.lineup.Queue(AlertRail, c); err == nil {
			d.lineup = next
		}
	}"""
assert old in s, "mT0"
p.write_text(s.replace(old, new, 1))
