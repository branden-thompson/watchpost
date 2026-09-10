# 0.16.0 P3. Reverse the precedence in toPrepare: a report is sent to be
# composed while a hazard on the rail has not been.
#
# THIS ONE SURVIVED until a fifth property was written for it, and it is the
# more dangerous of the two precedence bugs. Preparation is the expensive step
# (~1 s of network) and an unready rail card BLOCKS the air entirely, so
# composing a report ahead of a waiting hazard does not reorder the reads — it
# holds the whole station silent while the warning queues behind a weather
# report. Every property that watched what took the air passed, because nothing
# took the air at all.
import pathlib
p = pathlib.Path("platform/lineup/lineup.go"); s = p.read_text()
old = "for _, t := range []Track{AlertRail, MainTrack} { // the precedence, in one line"
assert s.count(old) == 2, "m104"
lines = s.split("\n")
hits = [i for i, l in enumerate(lines) if old in l]
lines[hits[0]] = lines[hits[0]].replace("Track{AlertRail, MainTrack}", "Track{MainTrack, AlertRail}")
p.write_text("\n".join(lines))
