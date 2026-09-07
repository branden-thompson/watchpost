import pathlib
# The Director is never told the programme is RUNNING. It starts Stopped — a
# station comes up silent, deliberately — and Stop was the only power it heard,
# so advances(MainTrack) is false for the life of the process and the bed never
# moves on. Watchlist looks like it simply does nothing, on the relay path and
# the synth path alike.
#
# Found at UAT 2026-09-04, and no test caught it because every fixture sent
# Powered{Running} itself: the tests supplied what the wiring had forgotten.
p = pathlib.Path("app/radio.go"); s = p.read_text()
old = '	if was == "" && mode != "" {'
new = '	if false && was == "" && mode != "" {'
assert old in s, "mM3"
p.write_text(s.replace(old, new, 1))
