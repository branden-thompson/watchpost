import pathlib
# 0.19.0 D-210: targets with the same answer share one line.
# Killed by TestIdenticalTargetsShareALine.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\t\tif _, met := named[say]; !met {\n'
assert old in s, 'm407'
p.write_text(s.replace(old, '\t\tif _, met := named[say]; !met || i > 0 {\n'))
