import pathlib
# W2.5 (FR-11.4, D-81): the terminal cursor stands on its first line; 0.19.3 D-362 places it there as a window's first words.
# Killed by TestTheCursorIsPlacedOnTheAcknowledgement.
p = pathlib.Path('modes/tty/cursor.go'); s = p.read_text()
old = "\treturn firstWords(lines)\n}\n\n// setupCursor"
assert s.count(old) == 1, 'm133'
p.write_text(s.replace(old, "\treturn 0, 0, false\n}\n\n// setupCursor", 1))
