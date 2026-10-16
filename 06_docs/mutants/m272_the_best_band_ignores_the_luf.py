import pathlib
# 0.19.0 D-197: the best band's lower edge is above the LUF.
# Killed by TestTheBestFrequencyIsNamed.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'b.low <= mhz && b.low >= l.luf {'
assert old in s, 'm272'
p.write_text(s.replace(old, 'b.low <= mhz {'))
