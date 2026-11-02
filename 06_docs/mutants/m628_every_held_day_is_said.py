import pathlib
# 0.19.3 D-363: only what the moment shown draws is said.
# Killed by TestEachLayerDrawnAddsItsSentence.
p = pathlib.Path('modes/tty/map_layer_words.go'); s = p.read_text()
old = "\treturn key, o.During.Meets(from, to)\n"
assert s.count(old) == 1, 'm628'
p.write_text(s.replace(old, "\treturn key, o.During.Meets(from, to) || true\n", 1))
