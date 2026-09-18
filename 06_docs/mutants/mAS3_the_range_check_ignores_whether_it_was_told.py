import pathlib
# `inServiceRange` stops asking whether the bounds are known and just compares.
# With nothing handed over that is `v >= 0 && v <= 0`, so only a radius of zero
# passes — and zero is the value the window uses for "not a number". A setting
# that accepts exactly the one value meaning "nothing".
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = "	lo, hi, ok := d.serviceBounds()\n	return ok && v >= lo && v <= hi"
new = "	lo, hi, _ := d.serviceBounds()\n	return v >= lo && v <= hi"
assert old in s, "mAS3"
p.write_text(s.replace(old, new, 1))
