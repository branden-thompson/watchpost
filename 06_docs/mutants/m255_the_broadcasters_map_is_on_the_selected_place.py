import pathlib
# 0.19.0 D-195: the Broadcaster's map is on its station.
# Killed by TestTheConsoleOpensTheMapOnTheStation.
p = pathlib.Path('modes/tty/map_role.go'); s = p.read_text()
old = '\tif d.mapRole().place == mapOnStation {\n'
assert old in s, 'm255'
p.write_text(s.replace(old, '\tif false {\n'))
