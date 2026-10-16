import pathlib
# 0.19.0 D-109: a storm is G 1 or more, not R or S.
# Killed by TestAStormSaysTypicalMayNotHold.
p = pathlib.Path('modes/tty/map_prop_scales.go'); s = p.read_text()
old = 'func propStorm(s ionomaps.Scales) bool { return raised(s.G) }'
assert old in s, 'm362'
p.write_text(s.replace(old, 'func propStorm(s ionomaps.Scales) bool { return s.G != 0 }'))
