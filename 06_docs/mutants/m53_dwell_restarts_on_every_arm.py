import pathlib
# The turn restarts on every status report. A live relay sends one each time its
# title changes and every one says Playing, so the countdown never elapses and
# the rotation stops dead on whichever station talks most.
#
# RE-ANCHORED (T3.2b): the rule was armDwell's idempotence in the deck; it is the
# Director's now, and it was very nearly lost in the move — the corpus guard
# caught this mutant having nothing left to anchor to, which is how the gap was
# found.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
start = s.index("	if d.bed.ref == ev.Ref && d.bed.live == ev.Live && !d.bed.since.IsZero() {")
end = s.index("	}\n", start) + 3
assert start < end, "m53"
p.write_text(s[:start] + s[end:])
