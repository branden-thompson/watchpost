import pathlib
# 0.19.0 D-193, A-51: under a long view name the answers keep a short indent, in the block's rows.
# Killed by TestThePropagationRowsFollowTheMock.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tif render.Width(name) > propPlaceColMax {'
assert old in s, 'm250'
p.write_text(s.replace(old, '\tif false && render.Width(name) > propPlaceColMax {'))
