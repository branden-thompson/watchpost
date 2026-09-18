import pathlib
# CANCEL MUST NOT SEND. `send` is tea.Program.Send and `Cancel` runs on
# Bubbletea's update loop, so a message posted from here goes into a channel
# that only the blocked loop can drain: the app freezes, no key works, and the
# listener has to kill the terminal.
#
# Softlocked at UAT 2026-09-03 by [space], [space], [esc] — and then by any [esc]
# with a read in progress. clearMarkUnlessReplaced is where this send belongs,
# because it runs on the READ's goroutine.
p = pathlib.Path("app/severe_read.go"); s = p.read_text()
old = """	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}"""
new = """	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if r.send != nil {
		r.send(tty.SevereReadingMsg{})
	}
}"""
assert s.count(old) == 1, "mJ4"
p.write_text(s.replace(old, new, 1))
