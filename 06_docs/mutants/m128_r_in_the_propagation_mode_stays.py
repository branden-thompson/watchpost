import pathlib
# W2.1 (D-153): R leaves the Propagation mode for Radar.
# Killed by TestThePropagationModeIsAKeymapAction.
p = pathlib.Path("modes/tty/map_temp.go"); s = p.read_text()
old = '\t\tmm.mapPane.prop = false\n\t\tmm, cmd := mm.enterMode(e)'
assert old in s, "m128"
p.write_text(s.replace(old, '\t\tmm, cmd := mm.enterMode(e)'))
