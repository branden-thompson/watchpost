import pathlib
# 0.19.0 D-210: a name with a comma keeps its own line.
# Killed by TestAFoundPlaceIsAnsweredAndNotKept.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tif strings.Contains(names[i], ",") {\n'
assert old in s, 'm443'
p.write_text(s.replace(old, '\t\tif i == 0 {\n'))
