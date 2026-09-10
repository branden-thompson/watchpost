# 0.14.0 DR-3, re-pinned at 0.16.0 P3. Reverse the precedence in Next: the
# programme takes the air while a hazard waits on the rail.
#
# THE ALERT RAIL DRAINS FIRST. This is the safety rule the two-track model
# exists for, and the property test asserts it at the MOMENT a card takes the
# air rather than by inspecting where things ended up.
import pathlib
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = "for _, t := range []Track{AlertRail, MainTrack} { // the precedence, in one line"
assert s.count(old) == 2, "m103"
lines = s.split("\n")
hits = [i for i, l in enumerate(lines) if old in l]
lines[hits[1]] = lines[hits[1]].replace("Track{AlertRail, MainTrack}", "Track{MainTrack, AlertRail}")
p.write_text("\n".join(lines))
