import pathlib
# 0.19.0 D-197: the best frequency is 85% of the upper limit.
# Killed by TestTheBestFrequencyIsNamed.
p = pathlib.Path('modes/tty/map_prop_origin.go'); s = p.read_text()
old = 'const propBestFactor = 0.85'
assert old in s, 'm271'
p.write_text(s.replace(old, 'const propBestFactor = 1.0'))
