import pathlib
# The grouping "simplified" away in dispatch: every effect of a step joins one
# serial run, so the next card\'s 1.03 s build waits behind the whole of this
# card\'s read. The cue still precedes the words, which is the point — this is
# the mutant the control test exists for.
#
# RE-ANCHORED at F-D2 round 2: byResource became runsOf.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = "\tfor _, group := range runsOf(fx) { // bounded by the step\'s effects (P10-02)"
new = "\tfor _, group := range [][]lineup.Effect{fx} { // bounded by the step\'s effects (P10-02)"
assert old in s, "mB2"
p.write_text(s.replace(old, new, 1))
