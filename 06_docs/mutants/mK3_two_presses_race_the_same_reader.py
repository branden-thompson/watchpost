import pathlib
# The press gate is removed, so two [space] presses — each on its own tea.Cmd
# goroutine — decide against snapshots the other has already invalidated. The
# loser pauses the read the winner just started and marks the row that is not
# reading. Invisible to the race detector: every access is locked, and it is the
# DECISION that races.
p = pathlib.Path("app/severe_read.go"); s = p.read_text()
old = "	r.press.Lock()\n	defer r.press.Unlock()\n\n"
assert old in s, "mK3"
p.write_text(s.replace(old, "", 1))
