import pathlib
# 0.19.5 Q2.8 (RN-12): a layer says its best answer, whichever came first.
# Killed by TestEachLayerSaysItsBestAnswer.
p = pathlib.Path('modes/tty/map_layer_words.go'); s = p.read_text()
old = '\t\tif cur, seen := best[key]; !seen || betterAnswer(a, cur) {\n'
assert s.count(old) == 1, 'm700'
p.write_text(s.replace(old, '\t\tif _, seen := best[key]; !seen {\n', 1))
