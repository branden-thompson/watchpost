import pathlib
# The relay-fault window drops out of the tick predicate, so no tick is armed
# while it is open: its countdown never steps, the fall-through never fires, and
# the frame never redraws between key presses. The window looks dead in the hand
# while every unit test of its clock and its nav stays green — which is exactly
# how it shipped (UAT 2026-09-05).
p = pathlib.Path("modes/tty/dashboard.go"); s = p.read_text()
old = "\tcase d.modal == modalRelayFault:\n\t\treturn true\n"
assert old in s, "mV0"
p.write_text(s.replace(old, "", 1))
