import pathlib
# 0.19.5 Q2.8 (RN-12): an answer with data is said before one without.
# Killed by TestEachLayerSaysItsBestAnswer.
p = pathlib.Path('modes/tty/map_layer_words.go'); s = p.read_text()
old = '\tif a.NoData != than.NoData {\n\t\treturn !a.NoData\n'
assert s.count(old) == 1, 'm701'
p.write_text(s.replace(old, '\tif a.NoData != than.NoData {\n\t\treturn a.NoData\n', 1))
