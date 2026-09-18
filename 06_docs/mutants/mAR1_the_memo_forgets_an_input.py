import pathlib
# The key drops `selected`, so the memo replays the tables with the OLD focus
# row: the operator presses ↓, the pointer does not move, and the model
# underneath is perfectly correct. That is F-30 exactly — the defect that froze
# three of Observer's windows in one release and took three UAT rounds to find
# the first one.
p = pathlib.Path("modes/tty/broadcaster_memo.go"); s = p.read_text()
old = "selected: b.selected,"
assert old in s, "mAR1"
p.write_text(s.replace(old, "", 1))
