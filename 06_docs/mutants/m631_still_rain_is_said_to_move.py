import pathlib
# 0.19.3 D-363: the radar's motion is said only while the heavier rain moves.
# Killed by TestTheRadarSaysItsMotion.
p = pathlib.Path('modes/tty/map_layer_words.go'); s = p.read_text()
old = "\t\tif !m.Moving || m.Missing != 0 {\n"
assert s.count(old) == 1, 'm631'
p.write_text(s.replace(old, "\t\tif m.Missing != 0 {\n", 1))
