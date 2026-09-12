import pathlib
# The card's STATUS line goes blank for every state, so the operator cannot tell
# a card that is READING from one that is ready from one still waiting on data —
# which is the first of the three questions D-87 says a card exists to answer.
p = pathlib.Path("modes/tty/broadcaster_manifest.go"); s = p.read_text()
old = "\tswitch c.State {\n\tcase lineup.OnAir:"
new = "\tswitch lineup.Discarded {\n\tcase lineup.OnAir:"
assert old in s, "mS2"
p.write_text(s.replace(old, new, 1))
