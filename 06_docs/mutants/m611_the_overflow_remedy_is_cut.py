import pathlib
# 0.19.3 D-365: the overflow line names its remedy whole, in the rows it needs.
# Killed by TestTheOverflowLineNamesItsRemedy.
p = pathlib.Path('modes/tty/map_boxes.go'); s = p.read_text()
old = '\t\tcontent = append(content[:max(most-len(tail), 1)], tail...)\n'
assert s.count(old) == 1, 'm611'
p.write_text(s.replace(old, '\t\tcontent = append(content[:max(most-len(tail), 1)], tail[0])\n', 1))
