import pathlib
# 0.19.0 D-193: with the LUF at or over the MUF no frequency serves.
# Killed by TestTheHeldFrequencyIsJudgedOnThePath.
p = pathlib.Path('platform/propanswers/limits.go'); s = p.read_text()
old = '\tcase l.Upper > 0 && l.LUF >= l.Upper:\n'
assert old in s, 'm242'
p.write_text(s.replace(old, '\tcase l.Upper > 0 && l.LUF > l.Upper:\n'))
