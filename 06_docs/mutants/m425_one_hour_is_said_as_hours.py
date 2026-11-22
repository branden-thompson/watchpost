import pathlib
# 0.19.0 one hour ahead is said "1 hour" (through plural, since 0.19.5's minor list).
# Killed by TestAnHourIsSaidSingly.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = 'func hoursWords(k int) string { return strconv.Itoa(k) + " hour" + frame.Plural(k) }'
assert s.count(old) == 1, 'm425'
s = s.replace(old, 'func hoursWords(k int) string { return strconv.Itoa(k) + " hours" }', 1)
# The line was the file's one use of internal/frame, so the import goes with it.
imp = '\t"github.com/branden-thompson/watchpost/modes/tty/internal/frame"\n'
assert s.count(imp) == 1 and s.count('frame.') == 0, 'm425 import'
p.write_text(s.replace(imp, ''))
