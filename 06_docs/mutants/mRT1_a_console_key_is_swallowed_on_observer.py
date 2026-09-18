import pathlib
# D-159, the fall-through contract. `keyAction` answers `[r]` on OBSERVER, where
# the key is the listener's REPEAT — so the console's request window steals a
# control from the surface the operator is actually looking at, and the
# listener's repeat simply stops working.
#
# THIS IS THE REGRESSION THE EXTRACTION COULD HAVE INTRODUCED. `update` was
# cyclomatic 43 and its switch was lifted out whole; several cases deliberately
# do NOT return, and falling past the switch is how the key reaches the active
# surface. Turning one of those into a return is invisible to every test that
# only checks the console still works.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	case actRequest:
		if r.consoleOwnsTheKeys() {
			r.observer = r.observer.openRequest()
			return r, nil, true
		}"""
new = """	case actRequest:
		if r.consoleOwnsTheKeys() {
			r.observer = r.observer.openRequest()
		}
		return r, nil, true"""
assert old in s, "mRT1"
p.write_text(s.replace(old, new, 1))
