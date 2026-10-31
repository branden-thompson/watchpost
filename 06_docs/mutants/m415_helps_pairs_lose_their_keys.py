import pathlib
# 0.19.0 AX-6: each key beside its name in a row of several.
# Killed by TestHelpFitsWithThePropagationKeys.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = 'out = append(out, shiftArrows(strings.Join(ks, "/"), ascii)+" "+n.name)'
assert old in s, 'm415'
p.write_text(s.replace(old, 'out = append(out, n.name)'))
