import pathlib
# D-154. The repeat guard is written for the AIR and eats the FENCE riding the
# same event. The operator narrows the service radius and then keys the surface
# they are already on — not a no-op — and the whole event is dropped, so a
# 100-mile hazard survives a narrowing to 25. That is the sentence `Aired.Fence`
# own comment claims to have fixed.
p = pathlib.Path("platform/lineup/air.go"); s = p.read_text()
old = """	if d.air == ev.To {
		return d.onRefenced(Refenced{Fence: ev.Fence})
	}"""
new = """	if d.air == ev.To {
		return d, nil
	}"""
assert old in s, "mRF1"
p.write_text(s.replace(old, new, 1))
