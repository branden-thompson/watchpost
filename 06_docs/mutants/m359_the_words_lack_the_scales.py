import pathlib
# 0.19.0 FR-1.18: the scales are named in the words' status block.
# Killed by TestARaisedScaleIsNamed.
p = pathlib.Path('modes/tty/map_prop_words.go'); s = p.read_text()
old = '\tif sw := propScalesWords(snap.Scales); sw != "" {\n'
assert old in s, 'm359'
p.write_text(s.replace(old, '\tif sw := propScalesWords(snap.Scales); false && sw != "" {\n'))
