import pathlib
# 0.19.5 Q3.3 (PF-3): the frame resolves its layout once; the layout-once pin is absolute now that a layout is cheaper than the frame's own work.
# Killed by TestLayoutOncePerViewAndKey.
p = pathlib.Path('modes/tty/view.go'); s = p.read_text()
old = '\tfl := d.layout()                 // once per frame (Q3, L5-F6)\n'
assert s.count(old) == 1, 'm737'
p.write_text(s.replace(old, '\t_ = d.layout()\n\tfl := d.layout()                 // once per frame (Q3, L5-F6)\n', 1))
