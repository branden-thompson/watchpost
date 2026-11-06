import pathlib
# 0.19.5 Q2.8 (RN-12): of two readings the nearer is said.
# Killed by TestEachLayerSaysItsBestAnswer.
p = pathlib.Path('modes/tty/map_layer_words.go'); s = p.read_text()
old = '\treturn a.Km < than.Km\n'
assert s.count(old) == 1, 'm702'
p.write_text(s.replace(old, '\treturn a.Km > than.Km\n', 1))
