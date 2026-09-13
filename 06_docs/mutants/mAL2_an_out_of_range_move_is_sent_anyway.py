import pathlib
# A position the running order does not have is sent to the schedule, which
# refuses it OUT OF SIGHT — FR-3.3's own rule is that an action is never shown as
# taken unless the schedule took it, and the operator is shown a closed window
# and an unmoved card (D-118).
p = pathlib.Path("modes/tty/broadcaster_manage.go"); s = p.read_text()
old = """	if err != nil || n < bcScheduledFrom || n > MainTrackSlots-1 {
		return 0
	}"""
assert old in s, "mAL2"
p.write_text(s.replace(old, """	if err != nil {
		return 0
	}""", 1))
