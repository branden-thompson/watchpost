import pathlib
# The window's close hook waits for the read's goroutine instead of just
# cancelling it. That wait runs on Bubbletea's UPDATE goroutine, so [esc] costs
# about half a second before the frame redraws — measured at UAT and called out
# as a priority issue: a TUI that hesitates on a key has given up the only thing
# it has over a browser.
p = pathlib.Path("app/dashboard.go"); s = p.read_text()
old = "			lp.reader.Cancel() // a keypress never waits: the frame redraws now"
new = "			lp.reader.End()"
assert old in s, "mJ2"
p.write_text(s.replace(old, new, 1))
