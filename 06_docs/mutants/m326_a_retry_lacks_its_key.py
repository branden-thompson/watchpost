import pathlib
# 0.19.0 D-120: a failed update is said with the key that asks again.
# Killed by TestAFailedFetchIsSaidWithItsPath.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = ' + " · " + d.firstKey(actMapRefresh) + " refreshes now"'
assert old in s, 'm326'
p.write_text(s.replace(old, ' + ""'))
