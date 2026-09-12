import pathlib
# Tab walks to the next group without asking whether this surface draws it, so on
# the console it lands in ALERTS - EVENTS or RELAY REPLAY — groups with no rows on
# screen. The focus is then somewhere the operator cannot see and the keyboard
# reads as dead (D-92).
p = pathlib.Path("modes/tty/setup.go"); s = p.read_text()
old = "stepGroup(d.setup.focus, 1, d.rowVisible)"
new = "stepGroup(d.setup.focus, 1, func(setupRowID) bool { return true })"
assert old in s, "mAA4"
p.write_text(s.replace(old, new, 1))
