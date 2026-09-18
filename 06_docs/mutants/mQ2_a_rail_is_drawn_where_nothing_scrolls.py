import pathlib
# The mark column is the wall on every row again, so a second vertical is drawn
# beside LIVE and UP NEXT — two regions with nothing to scroll. It reads as a
# column that stopped rather than as one that was never there, and the HUM LEAD
# reported it as "extra lines on the right side of the UI".
#
# RE-ANCHORED AT D-106. The guard used to be `if rail`, over a region that either
# had a control or did not; it is now `i >= from`, because the control belongs to
# the POOL and starts partway down the frame. Same rule, one seam along.
p = pathlib.Path("modes/tty/broadcaster_rail.go"); s = p.read_text()
old = "\t\tmarks[i] = \" \"\n\t\tif from >= 0 && i >= from {\n\t\t\tmarks[i] = g.Rail\n\t\t}\n"
new = "\t\tmarks[i] = g.Rail\n"
assert old in s, "mQ2"
p.write_text(s.replace(old, new, 1))
