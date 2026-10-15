import pathlib
# 0.19.0 D-193: Observer explores from the map's centre.
# Killed by TestTheOriginFollowsTheUIMode.
p = pathlib.Path('modes/tty/map_role.go'); s = p.read_text()
old = 'prop: propRole{origin: propFromCentre}},'
assert old in s, 'm237'
p.write_text(s.replace(old, 'prop: propRole{origin: propFromStation}},'))
