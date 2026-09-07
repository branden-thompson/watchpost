import pathlib
# The readers are told FIRST, so a subscriber decides what to pre-load from a
# lineup the same step is still changing.
#
# RE-ANCHORED at F-D1 round 2: the one-voice check sits between these two lines.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """\tfx = append(fx, Publish{Lineup: d.lineup})"""
new = """\tfx = append([]Effect{Publish{Lineup: d.lineup}}, fx...)"""
assert old in s, "m95"
p.write_text(s.replace(old, new, 1))
