import pathlib
# 0.19.0 FR-1.18 (D-88): a missing scales feed is said in the chart.
# Killed by TestMissingScalesAreNotShownAsZero.
p = pathlib.Path('modes/tty/map_prop_scales.go'); s = p.read_text()
old = '\t\treturn "SPACE WEATHER: " + scaleNotRead\n'
assert old in s, 'm355'
p.write_text(s.replace(old, '\t\treturn ""\n'))
