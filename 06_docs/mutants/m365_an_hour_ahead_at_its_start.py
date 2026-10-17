import pathlib
# 0.19.0 D-174: an hour ahead's night is for the middle of the hour.
# Killed by TestTheTerminatorIsDrawnForTheFieldsTime.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\t\treturn shown.Now.At.Add(30 * time.Minute).UTC()\n'
assert old in s, 'm365'
p.write_text(s.replace(old, '\t\treturn shown.Now.At.UTC()\n'))
