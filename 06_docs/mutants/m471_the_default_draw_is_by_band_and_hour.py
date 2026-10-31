import pathlib
# 0.19.0 W10.3 (D-224): the graded run's default draw is by band alone.
# Killed by TestTheDefaultDrawIsByBand.
p = pathlib.Path('tools/reachscore/score.go'); s = p.read_text()
old = 'fs.StringVar(&f.strata, "strata", strataBand, '
assert old in s, 'm471'
p.write_text(s.replace(old, 'fs.StringVar(&f.strata, "strata", strataBandHour, ', 1))
