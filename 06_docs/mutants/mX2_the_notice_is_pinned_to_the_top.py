import pathlib
# The standby notice stops being centred vertically and sits on the first row of the
# box, where it reads as a HEADING for contents that are not there rather than as
# the state of an empty slot. The HUM LEAD asked for "a centered text".
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\tbody[(len(body)-1)/2] = centerText(bcStandbyNotice, l.inner())"
new = "\tbody[0] = centerText(bcStandbyNotice, l.inner())"
assert old in s, "mX2"
p.write_text(s.replace(old, new, 1))
