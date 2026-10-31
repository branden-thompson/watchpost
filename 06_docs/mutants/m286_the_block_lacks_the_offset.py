import pathlib
# 0.19.0 FR-5.4: the rows under the map say the offset's state.
# Killed by TestAnUnusualOffsetIsSaid.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\tif n := propOffsetNote(snap.Offset); n != "" { // FR-5.4'
assert old in s, 'm286'
p.write_text(s.replace(old, '\tif n := propOffsetNote(snap.Offset); false && n != "" { // FR-5.4'))
