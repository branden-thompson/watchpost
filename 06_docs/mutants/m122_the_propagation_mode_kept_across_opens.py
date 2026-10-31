import pathlib
# W2.1 (D-154): every open is a weather mode; the Propagation mode is never kept.
# Killed by TestThePropagationModeIsAKeymapAction.
p = pathlib.Path("modes/tty/map_pane.go"); s = p.read_text()
old = "\td.mapPane.prop = false // every open is a weather mode"
assert old in s, "m122"
p.write_text(s.replace(old, "\t// every open is a weather mode"))
