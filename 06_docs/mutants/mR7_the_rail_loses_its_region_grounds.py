import pathlib
# The left rail stops naming its region in colour, so LIVE, UP NEXT and the queue
# are told apart only by letters read vertically one character at a time.
p = pathlib.Path("modes/tty/broadcaster_rail.go"); s = p.read_text()
old = "\treturn railColumnToned(label, rows, g, railTone(label))"
new = "\treturn railColumnToned(label, rows, g, \"\")"
assert old in s, "mR7"
p.write_text(s.replace(old, new, 1))
