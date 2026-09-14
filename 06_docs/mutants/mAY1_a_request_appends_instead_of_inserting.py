import pathlib
# The operator's request is APPENDED rather than inserted, so it lands at the
# bottom of the running order whatever position they asked for. PRIORITIZE then
# means "last", which is the opposite of what the word and the control say — and
# the modal would report a scheduled card the operator will not hear for an hour.
p = pathlib.Path("platform/lineup/operator.go"); s = p.read_text()
old = "	track = append(track[:at], append([]Card{c}, track[at:]...)...)"
assert old in s, "mAY1"
p.write_text(s.replace(old, "	track = append(track, c)", 1))
