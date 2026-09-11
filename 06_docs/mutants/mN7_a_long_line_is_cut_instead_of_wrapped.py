import pathlib
# The window truncates each line to the card instead of wrapping it, so the
# operator reads the first half of five sentences rather than the opening of the
# report. A report's sentences are routinely longer than a card is wide.
p = pathlib.Path("modes/tty/broadcaster_slots.go"); s = p.read_text()
old = "\t\twrapped = render.WrapLines(lines, room)"
new = "\t\tfor _, l := range lines {\n\t\t\twrapped = append(wrapped, render.TruncateCells(l, room))\n\t\t}"
assert old in s, "mN7"
p.write_text(s.replace(old, new, 1))
