import pathlib
# 0.19.0 FR-8.2 (#12): the temperature key carries the rain's density.
# Killed by TestTheAppMemoKeysCoverTheirInputs.
p = pathlib.Path('app/maptemp.go'); s = p.read_text()
old = 'fahrenheit: ask.Fahrenheit, rainFull: ask.RainFull,'
assert old in s, 'm380'
p.write_text(s.replace(old, 'fahrenheit: ask.Fahrenheit,'))
