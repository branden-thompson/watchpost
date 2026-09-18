import pathlib
# D-135. The swap drops out of the help on Observer, where the Router owns it and
# neither surface's keymap carries it — so the one window a lost operator opens
# does not say how to reach the other surface at all: "it doesnt show the user
# how to swap between Observer and Broadcaster" (HUM LEAD, UAT 2026-09-15).
p = pathlib.Path("modes/tty/help_about.go"); s = p.read_text()
old = """	for _, act := range []term.Action{actSwapObserver, actSwapBroadcaster} {
		if bind, ok := bc[act]; ok {
			out[act] = bind
		}
	}"""
new = """	_ = bc"""
assert old in s, "mCH3"
p.write_text(s.replace(old, new, 1))
