import pathlib
# The standby box draws short instead of a read card's full height, so the whole
# running order shunts up when the station goes on air and back down when it stops —
# at exactly the moment the operator is watching the line-up. It is the same reason
# readBody's height is fixed whether there is a script or not.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "\t\t\t\trows = append(rows, lane.standbyBox(bcReadCardRows)...)"
new = "\t\t\t\trows = append(rows, lane.standbyBox(bcFlatCardRows)...)"
assert old in s, "mX6"
p.write_text(s.replace(old, new, 1))
