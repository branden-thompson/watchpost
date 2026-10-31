import pathlib
# 0.19.0 D-193: on the Broadcaster the map's centre is a target, answered from the station.
# Killed by TestTheCentreIsAnsweredAsTheMapPans.
p = pathlib.Path('modes/tty/map_role.go'); s = p.read_text()
old = 'prop: propRole{origin: propFromStation, target: true}},'
assert old in s, 'm238'
p.write_text(s.replace(old, 'prop: propRole{origin: propFromStation}},'))
