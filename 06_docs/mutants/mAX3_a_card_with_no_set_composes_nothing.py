import pathlib
# A card that names no report set composes an EMPTY report instead of a full one.
# Every card in the tree is in that state — the Director's own cards carry no set
# — so the station would read a frame with nothing in it and the rotation would
# go quiet (R2).
#
# Re-pointed at R4b: the set now travels ON the effect rather than through a
# `wants` callback, so the default is an `Empty()` check rather than a nil hook.
# Same rule, and the mutation is the same lie: empty read as none.
p = pathlib.Path("app/schedule.go"); s = p.read_text()
old = """		if want.Empty() {
			want = report.Everything()
		}
"""
assert old in s, "mAX3"
p.write_text(s.replace(old, "", 1))
