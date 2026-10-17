import pathlib
# 0.19.0 W3.1 (FR-1.5): the Propagation mode's night is drawn for the field's time.
# Killed by TestTheTerminatorIsDrawnForTheFieldsTime.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\treturn shown.Now.At.UTC().Truncate(nightStep)\n'
assert old in s, 'm364'
p.write_text(s.replace(old, '\treturn d.now().UTC().Truncate(nightStep)\n'))
