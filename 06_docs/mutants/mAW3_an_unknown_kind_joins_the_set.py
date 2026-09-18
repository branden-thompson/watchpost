import pathlib
# `Add` stops checking the bound, so a kind this build does not know shifts into
# a bit nobody owns. A set written by a later version then reads as carrying
# reports it does not carry — and on a 32-bit set the shift is undefined past the
# width, so the bit lands somewhere arbitrary.
p = pathlib.Path("platform/report/report.go"); s = p.read_text()
old = """	if k >= numKinds {
		return s
	}
	return s | 1<<k"""
assert old in s, "mAW3"
p.write_text(s.replace(old, "	return s | 1<<k", 1))
