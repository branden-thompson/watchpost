# RE-ANCHORED AT T3.8's completion: a card's words are a SCRIPT in parts, not a
# string (MVS-D-77). Card.Text became Card.Words, and Speak/Built carry a Script.
# The rules are unchanged; only the words they are written against moved.
import pathlib
# takeTheAir looks PAST a head that is not ready for something that is, so a
# ready report jumps an alert still being built and the schedule reads out of
# order.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """	next, track, ok := d.lineup.Next()
	if !ok || next.State != Standby || next.Script.Empty() {
		return d, nil, false
	}"""
new = """	next, track, ok := d.lineup.Next()
	if !ok {
		return d, nil, false
	}
	if next.State != Standby || next.Script.Empty() {
		for _, t := range []Track{AlertRail, MainTrack} {
			for _, c := range d.lineup.Cards(t) {
				if c.State == Standby && c.Words() != "" {
					next, track, ok = c, t, true
				}
			}
		}
	}
	if next.State != Standby || next.Script.Empty() {
		return d, nil, false
	}"""
assert old in s, "mA8"
p.write_text(s.replace(old, new, 1))
