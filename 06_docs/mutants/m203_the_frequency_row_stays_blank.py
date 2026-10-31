import pathlib
# 0.19.0 D-179: the row between the mode and the place says the frequency held.
# Killed by TestYourFrequencyIsSaid.
p = pathlib.Path('modes/tty/map_prop_freq.go'); s = p.read_text()
old = '\tif d.mapPane.propFreqMHz == 0 {\n\t\treturn ""\n\t}\n\tline := "YOUR FREQUENCY "'
assert old in s, 'm203'
p.write_text(s.replace(old, '\tif d.mapPane.propFreqMHz >= 0 {\n\t\treturn ""\n\t}\n\tline := "YOUR FREQUENCY "'))
