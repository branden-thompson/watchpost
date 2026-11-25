import pathlib
# 0.19.0 FR-1.14: finding a place is a keymap action.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\t\tnd, w := mm.flashMapKey(act, e).openPropFind(e)\n\t\treturn nd, nil, w, true\n'
assert old in s, 'm392'
p.write_text(s.replace(old, '\t\tnd, w := mm.flashMapKey(act, e), mapWant{}\n\t\treturn nd, nil, w, true\n'))
