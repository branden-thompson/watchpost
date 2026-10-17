import pathlib
# 0.19.0 D-207: Automatic hands the depth back to the terminal.
# Killed by TestTheColourDepthIsASetting.
p = pathlib.Path('modes/tty/map_depth.go'); s = p.read_text()
old = '\t}\n\treturn tuimaps.TerminalDepth\n}\n'
assert old in s, 'm438'
p.write_text(s.replace(old, '\t}\n\treturn tuimaps.Truecolor\n}\n'))
