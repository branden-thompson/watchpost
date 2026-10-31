import pathlib
# 0.19.0 D-207: the map is hinted the Setting's depth each frame.
# Killed by TestTheColourDepthIsASetting.
p = pathlib.Path('modes/tty/map_pane.go'); s = p.read_text()
old = '\td.mapPane.call("ColourDepth", func() { m.ColourDepth(depth) })'
assert old in s, 'm437'
p.write_text(s.replace(old, '\td.mapPane.call("ColourDepth", func() { _ = depth })'))
