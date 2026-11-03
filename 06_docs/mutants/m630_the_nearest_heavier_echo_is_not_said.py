import pathlib
# 0.19.3 D-363: the radar says the nearest heavier echo.
# Killed by TestEachLayerDrawnAddsItsSentence.
p = pathlib.Path('modes/tty/map_layer_words.go'); s = p.read_text()
old = "\tif a.Heavier != \"\" {\n"
assert s.count(old) == 1, 'm630'
p.write_text(s.replace(old, "\tif false {\n", 1))
