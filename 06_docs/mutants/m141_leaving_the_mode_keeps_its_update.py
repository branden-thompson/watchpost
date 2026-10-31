import pathlib
# 0.19.0 FR-4.2: leaving the Propagation mode ends its update.
# Killed by TestLeavingTheModeCancelsItsUpdate.
p = pathlib.Path("modes/tty/map_prop.go"); s = p.read_text()
old = "\t\td = d.stopPropagation() // the Propagation mode's update ends with it (FR-4.2)"
assert old in s, "m141"
p.write_text(s.replace(old, "\t\t// the Propagation mode's update ends with it (FR-4.2)"))
