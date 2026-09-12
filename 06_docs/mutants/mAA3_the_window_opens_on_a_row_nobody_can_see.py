import pathlib
# The Settings window opens focused on row zero regardless of surface — and row
# zero is the listener's DEFAULT LOCATION, which the console does not draw. The
# operator opens Settings and the first keystroke appears to do nothing, because
# the focus is on a row that is not on screen (D-92).
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = """	if !d.rowVisible(d.setup.focus) {
		d.setup.focus = nextRow(d.setup.focus, d.rowVisible)
	}
"""
assert old in s, "mAA3"
p.write_text(s.replace(old, "", 1))
