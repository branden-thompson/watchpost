# The first alert is no longer rendered behind the burst head, so its render
# falls into the head's pause instead. Every OTHER boundary in the burst still
# overlaps correctly, which is what makes this one worth a mutant of its own: a
# pin that checked the per-event boundaries stayed green while the listener
# heard the head, a beat, then a second of nothing before the first alert.
# RE-ANCHORED AT T3.8: the rule MOVED to the Reader (app/read_script.go). One
# Reader now performs every card — the live takeover today, the Director's Speak
# executor at T3.10 — so the rule this guards has one home instead of two.
import pathlib
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = """	if i+1 < len(sc.Parts) { // render the NEXT while this one sounds
		n := sc.Parts[i+1]
		r.next, r.haveNext = timeRender(n.Kind.String(), func() (clip, bool) { return r.s.prepare(n.Text) })
	}
"""
assert old in s, "mG3"
p.write_text(s.replace(old, "", 1))
