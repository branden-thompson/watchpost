# The cue send deleted: the takeover reads with the band showing whatever it was
# showing before, so the listener hears a hazard the screen never names.
#
# RE-ANCHORED at T2.3: the band has one owner now, and the ticker writes through
# it rather than constructing the message itself.
# RE-ANCHORED AT T3.8: the rule MOVED to the Reader (app/read_script.go). One
# Reader now performs every card — the live takeover today, the Director's Speak
# executor at T3.10 — so the rule this guards has one home instead of two.
import pathlib
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = """	if p.Ref != "" && r.h.cue != nil {
		r.h.cue(p.Ref)
	}
"""
assert old in s, "m48"
p.write_text(s.replace(old, "", 1))
