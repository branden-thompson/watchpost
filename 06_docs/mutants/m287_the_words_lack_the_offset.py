import pathlib
# 0.19.0 FR-5.4: the status words say the offset's state.
# Killed by TestAnUnusualOffsetIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tif n := propOffsetNote(s.Offset); n != "" {\n\t\twords = append(words, n) // FR-5.4'
assert old in s, 'm287'
p.write_text(s.replace(old, '\tif n := propOffsetNote(s.Offset); false && n != "" {\n\t\twords = append(words, n) // FR-5.4'))
