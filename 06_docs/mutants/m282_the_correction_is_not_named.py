import pathlib
# 0.19.0 FR-5.4: the no-readings correction in use is named.
# Killed by TestAnUnusualOffsetIsSaid.
p = pathlib.Path('modes/tty/map_prop_update.go'); s = p.read_text()
old = '\tcase o.Corrected:\n\t\treturn "corrected'
assert old in s, 'm282'
p.write_text(s.replace(old, '\tcase false:\n\t\treturn "corrected'))
