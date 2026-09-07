# T3.1's defect, put back: a bound that cuts a burst WHILE IT IS BEING READ,
# rather than at admission. Whichever hazards sorted last are the ones silenced,
# and under sustained arrivals they are silenced permanently. Two red-team
# rounds found this shape and each fix moved the boundary instead of removing
# it — a per-lane floor, then a time budget, and a sweep still broke it at about
# 8.1 s a read.
# RE-ANCHORED: an air check was added at the top of the loop (mH2's rule), which
# now sits between the two lines this used to anchor on. The rule is unchanged —
# nothing may bound the read once the card is admitted.
# RE-ANCHORED AT T3.8: the rule MOVED to the Reader (app/read_script.go). One
# Reader now performs every card — the live takeover today, the Director's Speak
# executor at T3.10 — so the rule this guards has one home instead of two.
import pathlib
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = """	for i, p := range sc.Parts { // bounded by the script (P10-02)
		if !r.readPart(sc, i, p) {"""
new = """	for i, p := range sc.Parts { // bounded by the script (P10-02)
		if i >= 3 {
			break
		}
		if !r.readPart(sc, i, p) {"""
assert old in s, "mG0"
p.write_text(s.replace(old, new, 1))
