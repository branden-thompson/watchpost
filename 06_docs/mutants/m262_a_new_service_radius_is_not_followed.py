import pathlib
# 0.19.0 D-195: an open map follows the station saved in Settings.
# Killed by TestTheServiceAreaIsAlwaysDrawn.
p = pathlib.Path('modes/tty/setup.go'); s = p.read_text()
old = '\tif e := d.mapEnv(); moved && d.mapModel.mapPane.m != nil && d.mapModel.mapRole(e).zone {\n'
assert old in s, 'm262'
p.write_text(s.replace(old, '\tif e := d.mapEnv(); false {\n'))
