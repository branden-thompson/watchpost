import pathlib
# The map's size counts the Propagation mode's note row as drawn.
# Killed by TestTheNoteRowsAreCountedAsDrawn.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\treturn 1\n\t}\n\treturn len(d.noteLines(width))'
assert old in s, 'm174'
p.write_text(s.replace(old, '\t\treturn 2\n\t}\n\treturn len(d.noteLines(width))'))
