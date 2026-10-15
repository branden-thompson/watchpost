import pathlib
# 0.19.0 D-193: Observer explores from the map's centre.
# Killed by TestTheOriginFollowsTheUIMode.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'SurfaceObserver:    {origin: propFromCentre},'
assert old in s, 'm237'
p.write_text(s.replace(old, 'SurfaceObserver:    {origin: propFromStation},'))
