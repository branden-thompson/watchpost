import pathlib
# reads() forgets that a read can be ON THE AIR. It was decorative once — the
# pause carried its own copy of the rule in a branch of its own, and deleting
# this clause changed nothing — which is precisely how a "single enumeration"
# ends up with a third nobody asks. The pause has no second branch now, so this
# clause is the only thing that knows the air is a place a read can be.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = """	if j := d.onAir; live(j) && j.class == narrateRead {
		out = append(out, j)
	}
"""
assert old in s, "mL4"
p.write_text(s.replace(old, "", 1))
