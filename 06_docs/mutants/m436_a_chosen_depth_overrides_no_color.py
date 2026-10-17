import pathlib
# 0.19.0 D-207: with NO_COLOR set the map draws no colour, whatever is chosen.
# Killed by TestTheColourDepthIsASetting.
p = pathlib.Path('modes/tty/map_depth.go'); s = p.read_text()
old = '\tif d.cfg.NoColor {\n\t\treturn tuimaps.TerminalDepth\n'
assert old in s, 'm436'
p.write_text(s.replace(old, '\tif d.cfg.NoColor && false {\n\t\treturn tuimaps.TerminalDepth\n'))
