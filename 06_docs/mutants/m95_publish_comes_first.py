import pathlib
# The readers are told FIRST, so a subscriber decides what to pre-load from a
# lineup the same step is still changing.
#
# RE-ANCHORED at F-D1 round 2: the one-voice check sits between these two lines.
# RE-ANCHORED AGAIN 2026-09-10: the effect carries the POWER as well now (the
# console reads both from one publish), and the mutant had stopped applying — so
# the ordering rule it guards was UNMEASURED for the whole layout phase.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\tfx = append(fx, Publish{Lineup: d.lineup, Power: d.Power()})"""
new = """\tfx = append([]Effect{Publish{Lineup: d.lineup, Power: d.Power()}}, fx...)"""
assert old in s, "m95"
p.write_text(s.replace(old, new, 1))
