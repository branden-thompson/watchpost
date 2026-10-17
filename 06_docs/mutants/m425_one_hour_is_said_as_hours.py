import pathlib
# 0.19.0 one hour ahead is said "1 hour".
# Killed by TestAnHourIsSaidSingly.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\tif k == 1 {\n\t\treturn "1 hour"\n'
assert old in s, 'm425'
p.write_text(s.replace(old, '\tif k == -1 {\n\t\treturn "1 hour"\n'))
