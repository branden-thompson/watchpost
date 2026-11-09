import pathlib
# The Settings window opens focused on row zero without asking whether it is
# drawn. A focus on a row not on screen swallows the first keystroke (D-92).
#
# AN EQUIVALENT MUTANT since 0.18.0 D-70 (0.19.5 A-24): every surface draws
# every row, row zero included, so it survives. Kept, not retired: if a surface
# hides a row again, TestTheWindowOpensOnARowThisSurfaceDraws can catch it.
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = """	if !d.rowVisible(d.setup.focus) {
		d.setup.focus = nextRow(d.setup.focus, d.rowVisible)
	}
"""
assert old in s, "mAA3"
p.write_text(s.replace(old, "", 1))
