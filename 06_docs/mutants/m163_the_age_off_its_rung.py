import pathlib
# D-166: the snapshot's age on its rung of the freshness ramp.
# Killed by TestRoundFourMarksStandOut.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '" " + tint(ago, render.FreshnessTone(age)) + '
assert old in s, 'm163'
p.write_text(s.replace(old, '" " + tint(ago, render.FreshnessTone(0)) + '))
