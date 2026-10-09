import pathlib
# W2.5 (FR-11.1, D-81): the first P shows the acknowledgement before anything is fetched.
# Killed by TestTheAcknowledgementShowsBeforeTheFirstFetch.
p = pathlib.Path("modes/tty/map_prop.go"); s = p.read_text()
old = "\tif d.mapMode() == modePropagation && d.propAckDue() {"
assert old in s, "m129"
p.write_text(s.replace(old, "\tif d.mapMode() == modePropagation && d.propAckDue() && false {"))
