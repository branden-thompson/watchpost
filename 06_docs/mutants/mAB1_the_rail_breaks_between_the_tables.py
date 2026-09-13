import pathlib
# The scroll control goes back to leaving blank rows blank — which puts a hole in
# the rail on the one row between the running order and the pool, and a rail with
# a hole in it reads as two rails (D-104). One pointer, one control.
p = pathlib.Path("modes/tty/broadcaster_rail.go"); s = p.read_text()
old = """		mark := marks[i]
"""
assert old in s, "mAB1"
new = """		mark := marks[i]
		if strings.TrimSpace(r) == "" && mark != glyphs.Up && mark != glyphs.Down {
			mark = " "
		}
"""
s = s.replace(old, new, 1)
assert '"strings"' not in s, "mAB1: strings is back in the imports"
p.write_text(s.replace('import (\n', 'import (\n\t"strings"\n', 1))
