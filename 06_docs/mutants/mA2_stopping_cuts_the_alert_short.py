import pathlib
# The alert exemption narrowed to nothing — the condition kept so the tree still
# compiles, and made unreachable, which is how a rule usually stops applying.
# Stop now takes whatever is on the air off it, including an alert mid-read.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = "	if !held || track == AlertRail {"
new = "	if !held || track == numTracks {"
assert old in s, "mA2"
p.write_text(s.replace(old, new, 1))
