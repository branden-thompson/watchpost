import pathlib
# 0.19.0 P-1: an answer from or to the centre is asked once the view settles.
# Killed by TestAPanAsksOnceItSettles.
p = pathlib.Path('modes/tty/map_prop_chart.go'); s = p.read_text()
old = '\tmoving := d.mapPane.viewGen != d.mapPane.viewAsked\n'
assert old in s, 'm434'
p.write_text(s.replace(old, '\tmoving := false\n'))
