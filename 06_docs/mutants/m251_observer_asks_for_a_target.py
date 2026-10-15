import pathlib
# 0.19.0 D-193: on Observer the centre is the origin, never asked as a target.
# Killed by TestTheOriginFollowsTheUIMode.
p = pathlib.Path('modes/tty/map_prop_centre.go'); s = p.read_text()
old = 'if !ok || d.modal != modalMap || !d.propRole().target {'
assert old in s, 'm251'
p.write_text(s.replace(old, 'if !ok || d.modal != modalMap {'))
