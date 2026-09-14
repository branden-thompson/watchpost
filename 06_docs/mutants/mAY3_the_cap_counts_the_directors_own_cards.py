import pathlib
# The cap counts EVERY card on the track rather than the ones the operator can
# see, so the Director's structural cards — transitions, station credits — eat
# the running order's slots. The console then draws fewer reports than the
# schedule holds, and an insert sheds a card to make room for a transition the
# operator never asked about.
p = pathlib.Path("platform/lineup/operator.go"); s = p.read_text()
old = "	for out.visible(t) > MainTrackCap {"
assert old in s, "mAY3"
p.write_text(s.replace(old, "	for len(out.tracks[t]) > MainTrackCap {", 1))
