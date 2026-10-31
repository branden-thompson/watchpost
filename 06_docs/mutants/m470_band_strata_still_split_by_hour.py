import pathlib
# 0.19.0 W10.3 (D-219): strata by band alone keep the hours together.
# Killed by TestBandStrataKeepTheHoursSignal.
p = pathlib.Path('tools/reachscore/scenarios.go'); s = p.read_text()
old = '	if d.byHour {\n		hour = s.At.Hour()'
assert old in s, 'm470'
p.write_text(s.replace(old, '	if true {\n		hour = s.At.Hour()', 1))
