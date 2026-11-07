import pathlib
# 0.19.5 Q4.3 (PF-8): the tape memo keys on the model clock's day, since an alert's time reads as a date once its day has passed.
# Killed by TestTheTapeTurnsOverAtMidnight.
p = pathlib.Path('modes/tty/ticker.go'); s = p.read_text()
old = 'clock: o.Clock, year: now.Year(), day: now.YearDay()}'
assert s.count(old) == 1, 'm741'
p.write_text(s.replace(old, 'clock: o.Clock, year: now.Year()}', 1))
