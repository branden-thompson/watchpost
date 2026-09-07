import pathlib
# The dwell is applied to the synthesised broadcast as well as to a live relay.
# The synth broadcast ENDS ON ITS OWN and advances at its own end; giving it a
# turn as well cuts a location's report off partway through to move to the next.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
old = '	if d.settings.Dwell <= 0 || !d.bed.live || d.bed.ref == "" {'
new = '	if d.settings.Dwell <= 0 || d.bed.ref == "" {'
assert old in s, "mK1"
p.write_text(s.replace(old, new, 1))
