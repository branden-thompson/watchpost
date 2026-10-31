import pathlib
# W2.5 (FR-11.4, D-81): the terminal cursor stands on its first line.
# Killed by TestTheCursorIsPlacedOnTheAcknowledgement.
p = pathlib.Path("modes/tty/map_prop.go"); s = p.read_text()
old = "\tif d.modal != modalPropAck {\n\t\treturn 0, 0, false\n\t}\n\treturn propAckCursor(frame)"
assert old in s, "m133"
p.write_text(s.replace(old, "\treturn 0, 0, false"))
