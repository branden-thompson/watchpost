import pathlib
# 0.19.0 D-332: STALE at an hour whatever the refresh, so Hourly says it most of each hour.
# Killed by TestStaleFollowsTheRefresh.
p = pathlib.Path('modes/tty/map_prop_settings.go'); s = p.read_text()
old = 'func (r propRefresh) staleAfter() time.Duration { return time.Hour + r.every() }\n'
assert s.count(old) == 1, 'm589'
p.write_text(s.replace(old, 'func (r propRefresh) staleAfter() time.Duration { return time.Hour }\n', 1))
