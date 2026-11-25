import pathlib
# 0.19.0 FR-1.16: a frequency not taken is said, and the entry stays open.
# Killed by TestABadFrequencyIsSaidNotGuessed.
p = pathlib.Path('modes/tty/map_prop_freq.go'); s = p.read_text()
old = '\t\t\tmm.mapPane.propFreqErr = errFreqWords\n\t\t\treturn mm, nil, mapWant{}\n'
assert old in s, 'm205'
p.write_text(s.replace(old, '\t\t\treturn mm, nil, mapWant{Close: true}\n'))
