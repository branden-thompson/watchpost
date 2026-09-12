import pathlib
# The card window's generation moves on every hand-in whether anything changed or
# not. The refresh runs on EVERY update, so the memo key differs every frame and
# the window re-renders continuously for as long as it is open — which is the cost
# the comparison in showCard exists to avoid, and the reason it compares at one
# agreed Opts rather than trusting the caller.
p = pathlib.Path("modes/tty/broadcaster_detail.go"); s = p.read_text()
old = "	if d.cardID == id && d.cardRows != nil && sameCard(d.cardRows, rows, at) {"
new = "	if false {"
assert old in s, "mW6"
p.write_text(s.replace(old, new, 1))
