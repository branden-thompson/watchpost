import pathlib
# A synthesised broadcast that played to the end of its cycle does not move on,
# so a Watchlist rotation stops at whichever location finished first. The dwell
# cannot cover it: a relay never ends and the synth broadcast never dwells —
# advanceQueue had two callers and the absorb needs both.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
i = s.index("func (d Director) onEnded(Ended) (Director, []Effect) {")
j = s.index("\n}\n", i) + 3
assert i < j, "mM1"
p.write_text(s[:i] + "func (d Director) onEnded(Ended) (Director, []Effect) { return d, nil }\n" + s[j:])
