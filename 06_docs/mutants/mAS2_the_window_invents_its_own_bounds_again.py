import pathlib
# THE SECOND CARRIER COMES BACK. The window falls back to 2 and 100 of its own
# when the app has not told it — which looks harmless and undoes the whole of
# D-124: the numbers exist in two places again, and the day `platform/config`
# changes one the window goes on offering the old range and the storage clamps
# behind the operator's back.
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = """	lo, hi = d.cfg.ServiceRadiusMinMi, d.cfg.ServiceRadiusMaxMi
	return lo, hi, lo > 0 && hi >= lo"""
new = """	lo, hi = d.cfg.ServiceRadiusMinMi, d.cfg.ServiceRadiusMaxMi
	if lo <= 0 || hi < lo {
		lo, hi = 2, 100
	}
	return lo, hi, true"""
assert old in s, "mAS2"
p.write_text(s.replace(old, new, 1))
