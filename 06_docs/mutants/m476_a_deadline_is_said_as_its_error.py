import pathlib
# 0.19.0 BUILD red team round 2 (FR-5.2, BL2-7): an update out of time is said in words, never the context's error.
# Killed by TestADeadlineIsSaidInWords.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tcase errors.Is(err, context.DeadlineExceeded):'
assert old in s, 'm476'
p.write_text(s.replace(old, '\tcase false:', 1))
