import pathlib
# The cue reaches the band but leaves no record (DR-18): nothing a test or a
# diagnostic can read afterwards says it happened.
#
# RE-ANCHORED at T2.3 (the executor cues through the band's one owner), at
# T3.10b (the producer's record IS the event now), and at the T3.10 red team
# (runCue delegates to cueFor, so the record is the only thing left here).
p = pathlib.Path("app/executors.go"); s = p.read_text()
old = "\tx.band.note(lineup.Describe(v))\n\treturn nil\n}"
new = "\treturn nil\n}"
assert old in s, "mC2"
p.write_text(s.replace(old, new, 1))
