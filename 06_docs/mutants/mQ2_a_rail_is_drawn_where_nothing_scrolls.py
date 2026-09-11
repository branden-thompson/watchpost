import pathlib
# The mark column is the wall on every row again, so a second vertical is drawn
# beside LIVE and UP NEXT — two regions with nothing to scroll. It reads as a
# column that stopped rather than as one that was never there, and the HUM LEAD
# reported it as "extra lines on the right side of the UI".
p = pathlib.Path("modes/tty/broadcaster_rail.go"); s = p.read_text()
old = "\t\tmarks[i] = \" \"\n\t\tif rail {\n\t\t\tmarks[i] = g.Rail\n\t\t}\n"
new = "\t\tmarks[i] = g.Rail\n"
assert old in s, "mQ2"
p.write_text(s.replace(old, new, 1))
