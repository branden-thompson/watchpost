import pathlib
# FR-6.5 (0.19.0 W1.2): a record's values are copied to its reader, never the store's own.
# Killed by TestARecordReadIsItsReadersOwn.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "	return append([]float64(nil), v...)\n"
assert old in s, "m113"
p.write_text(s.replace(old, "	return []float64(v)\n"))
