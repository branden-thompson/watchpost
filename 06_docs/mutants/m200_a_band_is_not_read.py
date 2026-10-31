import pathlib
# 0.19.0 FR-1.16 (A-47): a band is read by its name.
# Killed by TestYourFrequencyIsSaid.
p = pathlib.Path('modes/tty/map_prop_freq.go'); s = p.read_text()
old = '\t\t\tif strings.TrimSuffix(b.name, "m") == metres {\n'
assert old in s, 'm200'
p.write_text(s.replace(old, '\t\t\tif b.name == metres {\n'))
