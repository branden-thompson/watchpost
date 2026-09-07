import pathlib
# The running state stops being read: the main track advances whatever the
# listener did, so a report comes up on a radio they stopped.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = "	return d.power == Running"
new = "	return d.power == Running || d.power == Stopped"
assert old in s, "mA0"
p.write_text(s.replace(old, new, 1))
