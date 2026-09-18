import pathlib
# The refresh walk goes back to the MAIN TRACK alone, so a card opened from the
# takeover box never updates. It was complete while a digit was the only way in;
# `[A]` opens a card on the ALERT RAIL, and a window that never refreshes goes
# stale exactly where staleness matters most — a burst gains hazards while the
# operator is reading it, and the window goes on showing the old list.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	if id, rows, ok := r.broadcaster.alertDetail(); ok && id == r.observer.cardID {
		r.observer = r.observer.showCard(id, rows, r.broadcaster.alertWindowGround(), r.observer.opts())
	}
"""
assert old in s, "mAV3"
p.write_text(s.replace(old, "", 1))
