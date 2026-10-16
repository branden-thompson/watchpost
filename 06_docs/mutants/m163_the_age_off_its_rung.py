import pathlib
# D-166: the snapshot's age on its rung of the freshness ramp.
# Killed by TestRoundFourMarksStandOut.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\ttone := render.FreshnessTone(age)\n'
assert old in s, 'm163'
p.write_text(s.replace(old, '\ttone := render.FreshnessTone(age * 0)\n'))
