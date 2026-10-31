import pathlib
# 0.19.0 FR-1.18 (D-88): a missing scales feed is said in the words.
# Killed by TestMissingScalesAreNotShownAsZero.
p = pathlib.Path('modes/tty/map_prop_scales.go'); s = p.read_text()
old = '\t\treturn "space weather: " + scaleNotRead\n'
assert old in s, 'm356'
p.write_text(s.replace(old, '\t\treturn ""\n'))
