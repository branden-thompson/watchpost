import pathlib
# 0.19.0 D-199: in Forecast mode the night is for the middle of the step shown.
# Killed by TestTheNightIsForTheStepShown.
p = pathlib.Path('modes/tty/map_night.go'); s = p.read_text()
old = 'sp.From.Add(sp.Until.Sub(sp.From) / 2).UTC()'
assert old in s, 'm304'
p.write_text(s.replace(old, 'sp.From.UTC()'))
