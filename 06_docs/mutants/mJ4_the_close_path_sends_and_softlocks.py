import pathlib
# The close path sends on the CALLER'S goroutine. `send` is tea.Program.Send and
# `close` runs on Bubbletea's update loop, so the message goes into a channel
# that only the blocked loop can drain: the app freezes, no key works, and the
# listener has to kill the terminal.
#
# Softlocked at UAT 2026-09-03 by [space], [space], [esc] — and then by any [esc]
# with a read in progress.
p = pathlib.Path("app/severe_read.go"); s = p.read_text()
old = """	_ = had
}"""
new = """	if had && r.send != nil {
		r.send(tty.SevereReadingMsg{})
	}
}"""
assert old in s, "mJ4"
p.write_text(s.replace(old, new, 1))
