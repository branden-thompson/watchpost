import pathlib
# 0.19.0 D-193, D-195: a UI mode with no role is Observer's.
# Killed by TestEveryUIModeHasAMapRole.
p = pathlib.Path('modes/tty/map_role.go'); s = p.read_text()
old = '\t\treturn mapRoles[SurfaceObserver]\n'
assert old in s, 'm239'
p.write_text(s.replace(old, '\t\treturn mapRoles[SurfaceBroadcaster]\n'))
