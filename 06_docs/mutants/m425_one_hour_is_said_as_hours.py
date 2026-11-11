import pathlib
# 0.19.0 one hour ahead is said "1 hour" (through plural, since 0.19.5's minor list).
# Killed by TestAnHourIsSaidSingly.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = 'func hoursWords(k int) string { return strconv.Itoa(k) + " hour" + plural(k) }'
assert s.count(old) == 1, 'm425'
p.write_text(s.replace(old, 'func hoursWords(k int) string { return strconv.Itoa(k) + " hours" }', 1))
