import pathlib
# 0.19.0 D-193: with the LUF at or over the MUF no frequency serves.
# Killed by TestTheHeldFrequencyIsJudgedOnThePath.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = '\tcase l.upper > 0 && l.luf >= l.upper:\n'
assert old in s, 'm242'
p.write_text(s.replace(old, '\tcase l.upper > 0 && l.luf > l.upper:\n'))
