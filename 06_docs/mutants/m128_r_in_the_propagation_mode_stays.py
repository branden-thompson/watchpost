import pathlib
# W2.1 (D-153): R leaves the Propagation mode for Radar.
# Killed by TestThePropagationModeIsAKeymapAction.
p = pathlib.Path("modes/tty/map_temp.go"); s = p.read_text()
old = "\t\td.mapPane.prop = false\n\t\td, cmd := d.enterMode()"
assert old in s, "m128"
p.write_text(s.replace(old, "\t\td, cmd := d.enterMode()"))
