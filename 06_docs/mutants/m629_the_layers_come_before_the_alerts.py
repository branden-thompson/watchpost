import pathlib
# 0.19.3 D-363: the layers' sentences follow the alerts.
# Killed by TestEachLayerDrawnAddsItsSentence.
p = pathlib.Path('modes/tty/map_describe.go'); s = p.read_text()
old = "\treturn append(append(out, said...), layers...)\n"
assert s.count(old) == 1, 'm629'
p.write_text(s.replace(old, "\treturn append(append(out, layers...), said...)\n", 1))
