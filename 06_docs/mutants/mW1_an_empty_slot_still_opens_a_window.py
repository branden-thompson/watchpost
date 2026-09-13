import pathlib
# A slot number with no card behind it opens the window anyway, onto nothing — and
# it CONSUMES the key, so the digit no longer falls through the way an unbound key
# does. The operator keys [7] on a station whose Director has filled two slots and
# gets an empty box: a window that asks them to read the absence of a report
# (broadcaster_detail.go, cardDetail).
#
# Re-pointed after `[A]` joined the digits at the same door: the refusal is now
# one check for both handles, which is what makes it worth keeping in one place.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	if !ok {
		return r, false
	}
	// THE SURFACE DOES NOT CHANGE"""
new = """	// THE SURFACE DOES NOT CHANGE"""
assert old in s, "mW1"
p.write_text(s.replace(old, new, 1))
