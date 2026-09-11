import pathlib
# The band is never given back.
#
# The shipped defect DR-24 was written about: the release was a closure's last
# statement with four early returns above it, each leaving the band holding a
# callout for a read that had stopped. It went unnoticed because the AUDIO is
# released unconditionally by the arbiter — the sound came back, so the station
# seemed fine, and only the band stayed wrong.
#
# RE-ANCHORED TWICE. At T3.10b the rule moved into Step's output; at D-82 the
# release gained the LANE it belongs to, so that a report's exit cannot clear a
# hazard's callout (F-71). This is still the one line that makes the pairing true.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\t\tfx = append(fx, ReleaseTicker{ID: id, Track: track})"
new = "\t\t_, _ = id, track"
assert old in s, "mR1"
p.write_text(s.replace(old, new, 1))
