import pathlib
# 0.19.0 FR-3.6: the words say when the refresh came.
# Killed by TestARefreshIsSaidOnce.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\t\td.mapPane.propUpdatedAt = d.now()\n'
assert old in s, 'm335'
p.write_text(s.replace(old, ''))
