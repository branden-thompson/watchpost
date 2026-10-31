import pathlib
# W2.2 (FR-1.7): with no bound, the Propagation mode's zoom stops at the whole world.
# Killed by TestThePropagationModePansAndZooms.
p = pathlib.Path("modes/tty/map_pane.go"); s = p.read_text()
old = "\tif _, z := m.Centre(); z < least {"
assert old in s, "m124"
p.write_text(s.replace(old, "\tif _, z := m.Centre(); z < least-100 {"))
