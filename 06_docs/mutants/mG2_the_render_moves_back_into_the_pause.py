# The pre-build is undone inside the read loop: each line renders at cue time
# again instead of during the line before it, so a render (~1 s) sits in every
# pause of the burst. Nothing sounds wrong on its own — the words are correct and
# in order — but the listener hears about two and a half seconds between alerts
# where the ruling says one. Heard on a real alert at UAT 2026-09-03.
#
# RE-ANCHORED: the first line now arrives already rendered from openBurst, so
# this deletes the LOOKAHEAD, which is the rule for every later line.
# RE-ANCHORED AT T3.8: the rule MOVED to the Reader (app/read_script.go). One
# Reader now performs every card — the live takeover today, the Director's Speak
# executor at T3.10 — so the rule this guards has one home instead of two.
import pathlib
p = pathlib.Path("app/read_script.go"); s = p.read_text()
old = """	if len(sc.Parts) > 0 {
		r.next, r.haveNext = timeRender(sc.Parts[0].Kind.String(), func() (clip, bool) { return r.s.prepare(sc.Parts[0].Text) })
	}
"""
assert old in s, "mG2"
p.write_text(s.replace(old, "", 1))
