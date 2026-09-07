# The cue moved BELOW the words, so the band promises a callout after the read
# has already started — today's accidental ordering, undone.
#
# RE-ANCHORED at T2.3 (the ticker cues through the band's one owner) and again
# at T3.3, where the line stopped being rendered at cue time and became a clip
# prepared during the sound before it. The RULE is unchanged: the band is told
# before the words are HEARD, which is `deliver`, not the render.
# RE-ANCHORED AT T3.8: the rule MOVED to the Reader (app/read_script.go). One
# Reader now performs every card — the live takeover today, the Director's Speak
# executor at T3.10 — so the rule this guards has one home instead of two.
import pathlib
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = """	if p.Ref != "" && r.h.cue != nil {
		r.h.cue(p.Ref)
	}
	cur, have := r.next, r.haveNext"""
new = """	cur, have := r.next, r.haveNext"""
assert old in s, "m49"
s = s.replace(old, new, 1)
tail = """	if p.Ref != "" && r.h.mark != nil {"""
assert tail in s, "m49 tail"
p.write_text(s.replace(tail, """	if p.Ref != "" && r.h.cue != nil {
		r.h.cue(p.Ref)
	}
""" + tail, 1))
