import pathlib
# The freshness rule dropped: a landslide four days ago outranks this morning's
# thunderstorm warning again, which is the ruling DR-12 exists to encode.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = """	if now.Sub(at) <= DisasterFreshWindow {
		return CloseBand
	}
	return RemainingBand"""
new = """	return CloseBand"""
assert old in s, "m73"
p.write_text(s.replace(old, new, 1))
