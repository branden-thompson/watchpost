import pathlib
# D-156. The arithmetic stays and the Router stops feeding it, so `liveOffset`
# reads zero — which is exactly the buggy answer, silently. This is the half a
# unit test cannot reach: calling `requestSchedule` directly passes with the
# wiring absent, because an unset field and a running station give the same
# number. Only the seam the KEY takes proves the offset crosses (P-1).
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = "\tr.observer.liveOffset = r.broadcaster.liveOffset()\n"
assert old in s, "mRQ2"
p.write_text(s.replace(old, "", 1))
