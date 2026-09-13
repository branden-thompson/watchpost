import pathlib
# The card's stamp goes back to the window's long form — "Saturday September 12,
# 2026 @ 16:02:45" — which is forty-two cells of a card that has about sixty, all
# of them saying something the operator already knows (D-110).
p = pathlib.Path("modes/tty/broadcaster_manifest.go"); s = p.read_text()
old = """	out := c.BuiltAt.Format("01/02/06  15:04:05")"""
assert old in s, "mAD5"
p.write_text(s.replace(old, """	out := c.BuiltAt.Format("Monday January 2, 2006 @ 15:04:05")""", 1))
