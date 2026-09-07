# An event read aloud is never marked seen, so a later takeover reads it again
# and the listener hears the same alert twice.
#
# RE-ANCHORED at T3.3: `now` is captured at cue time and, since the line is
# rendered ahead of its own turn, the seen-mark is its only remaining reader.
# Deleting the mark alone left `now` unused and the tree uncompilable, which is
# an INVALID verdict rather than evidence either way (D-3) — so the mutant
# deletes the RULE and keeps the identifier that holds the tree together.
# RE-ANCHORED AT T3.8: the rule MOVED to the Reader (app/read_script.go). One
# Reader now performs every card — the live takeover today, the Director's Speak
# executor at T3.10 — so the rule this guards has one home instead of two.
import pathlib
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = """	if p.Ref != "" && r.h.mark != nil {
		r.h.mark(p.Ref)
	}
"""
assert old in s, "m22"
p.write_text(s.replace(old, "", 1))
