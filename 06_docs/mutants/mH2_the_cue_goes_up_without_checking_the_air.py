# The air check before each callout is deleted. A cue is a promise that words are
# coming, and the RETRY path reaches the first one through a second render with
# no hold between — so a sequence that ended during that render leaves a breaking
# headline on the band for a read that never happens.
#
# holdRest's own check does not cover this door: it fires when a HOLD has nothing
# left to wait, and the retry has no hold at all. Found by review, on code that
# had just been fixed for the same defect through a different path.
# RE-ANCHORED AT T3.8: the rule MOVED to the Reader (app/read_script.go). One
# Reader now performs every card — the live takeover today, the Director's Speak
# executor at T3.10 — so the rule this guards has one home instead of two.
import pathlib
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = """	if !r.s.awaitAir(nil) {
		return false
	}
	if p.Ref != "" && r.h.cue != nil {"""
new = """	if p.Ref != "" && r.h.cue != nil {"""
assert old in s, "mH2"
p.write_text(s.replace(old, new, 1))
