import pathlib
# 0.19.0 BUILD red team round 2 (BL2-4): an hour ahead's night is drawn for its start, the moment of its field and words.
# Killed by TestTheTerminatorIsDrawnForTheFieldsTime.
p = pathlib.Path('modes/tty/map_prop_draw.go'); s = p.read_text()
old = '\treturn shown.Now.At.UTC().Truncate(nightStep)\n}'
assert old in s, 'm485'
p.write_text(s.replace(old, '\treturn shown.Now.At.Add(30 * time.Minute).UTC().Truncate(nightStep)\n}', 1))
