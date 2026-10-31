import pathlib
# 0.19.0 D-178: an empty Enter clears the frequency held.
# Killed by TestYourFrequencyIsSaid.
p = pathlib.Path('modes/tty/map_prop_freq.go'); s = p.read_text()
old = '\t\t\td.mapPane.propFreqMHz, d.mapPane.propFreqLabel = 0, ""\n'
assert old in s, 'm204'
p.write_text(s.replace(old, '\t\t\t_ = 0\n'))
