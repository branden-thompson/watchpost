import pathlib
# 0.19.0 W1.2: a month's roll-up parts are pruned once the days' retention passes, as 0.18.0's years were.
# Killed by TestADayIsRolledUpThenPruned.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "	if start, ok := rollupMonth(name); ok {\n		return start.AddDate(0, 1, 0), true\n	}"
assert old in s, "m112"
p.write_text(s.replace(old, "	if start, ok := rollupMonth(name); ok && false {\n		return start.AddDate(0, 1, 0), true\n	}"))
