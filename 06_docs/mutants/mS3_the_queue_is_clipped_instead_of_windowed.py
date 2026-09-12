import pathlib
# The running order stops being windowed to the room it has, so a line-up taller
# than the terminal runs past the frame's closing inset and off the bottom —
# which FR-7.3 calls a defect rather than a degradation. A card is a manifest
# since D-87, so ten slots need about ninety rows and the reference terminal has
# seventy-four: this is the ordinary case, not an edge one.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\t\tscroll = append([]string(nil), scroll[:room]...)"
new = "\t\tscroll = append([]string(nil), scroll...)"
assert old in s, "mS3"
p.write_text(s.replace(old, new, 1))
