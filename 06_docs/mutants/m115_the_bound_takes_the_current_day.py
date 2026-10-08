import pathlib
# D-143 (0.19.0 W1.2): the byte bound never removes the day being recorded.
# Killed by TestTheCurrentDayIsNeverRemovedForTheBound.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "\t\tif !f.at.Before(today) {"
assert old in s, "m115"
p.write_text(s.replace(old, "\t\tif false && !f.at.Before(today) {"))
