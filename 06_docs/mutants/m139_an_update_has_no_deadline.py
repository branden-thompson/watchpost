import pathlib
# 0.19.0 W4.1: an update ends at its own 60 s deadline.
# Killed by TestAnUpdateHasItsOwnDeadline.
p = pathlib.Path("domains/propagation/propagation.go"); s = p.read_text()
old = "\tuctx, cancel := context.WithTimeout(ctx, s.deadline)"
assert old in s, "m139"
p.write_text(s.replace(old, "\tuctx, cancel := context.WithCancel(ctx)"))
