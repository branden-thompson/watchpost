import pathlib
# A slot number with no card behind it opens the window anyway, onto nothing — and
# it CONSUMES the key, so the digit no longer falls through the way an unbound key
# does. The operator keys [7] on a station whose Director has filled two slots and
# gets an empty box: a window that asks them to read the absence of a report
# (broadcaster_detail.go, cardDetail).
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	id, rows, ok := r.broadcaster.cardDetail(int(key[0] - '0'))
	if !ok {
		return r, false
	}"""
new = """	id, rows, _ := r.broadcaster.cardDetail(int(key[0] - '0'))"""
assert old in s, "mW1"
p.write_text(s.replace(old, new, 1))
