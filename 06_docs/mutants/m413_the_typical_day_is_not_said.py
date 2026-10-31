import pathlib
# 0.19.0 AX-4: the local area's typical day is said after its line.
# Killed by TestTheChartIsReadableInASCII.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tif len(out) == 1 {\n\t\t\tout = append(out, said("typical"'
assert old in s, 'm413'
p.write_text(s.replace(old, '\t\tif len(out) == -1 {\n\t\t\tout = append(out, said("typical"'))
