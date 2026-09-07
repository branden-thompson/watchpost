import pathlib
# The band is never given back.
#
# The shipped defect DR-24 was written about: the release was a closure's last
# statement with four early returns above it, each leaving the band holding a
# callout for a read that had stopped. It went unnoticed because the AUDIO is
# released unconditionally by the arbiter — the sound came back, so the station
# seemed fine, and only the band stayed wrong.
#
# RE-ANCHORED AT T3.10b, WHERE THE RULE NOW LIVES. The takeover's own closure is
# gone: the read is an effect, and every exit from the air emits its release from
# the schedule itself. That is strictly better — the pairing is a property of
# Step's output rather than a discipline about call sites — and this is the one
# line that makes it true.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = "\t\tfx = append(fx, ReleaseTicker{ID: id})"
new = "\t\t_ = id"
assert old in s, "mR1"
p.write_text(s.replace(old, new, 1))
