import pathlib
# The window opens with no position chosen again, so a resolved location is still
# not schedulable and `enter` refuses with "Choose a position" — an extra decision
# on every request, for a field that has an obvious safe answer.
#
# HUM LEAD, 2026-09-14: "default to the bottom - position 15."
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = """		slot:   strconv.Itoa(lineup.MainTrackCap - 1),
"""
assert old in s, "mBC1"
p.write_text(s.replace(old, "", 1))
