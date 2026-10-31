import pathlib
# 0.19.0 D-197: the best band's lower edge is above the LUF.
# Killed by TestTheBestFrequencyIsNamed.
p = pathlib.Path('platform/propanswers/limits.go'); s = p.read_text()
old = 'low <= mhz && low >= l.LUF {'
assert old in s, 'm272'
p.write_text(s.replace(old, 'low <= mhz {'))
