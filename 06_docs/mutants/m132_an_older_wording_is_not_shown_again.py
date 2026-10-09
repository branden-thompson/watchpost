import pathlib
# W2.5 (FR-11.3, A-9): words seen older are shown once more.
# Killed by TestAChangedAcknowledgementShowsAgain.
p = pathlib.Path("modes/tty/map_prop_ack.go"); s = p.read_text()
old = "func ackDue(seen, current int) bool { return seen < current }"
assert old in s, "m132"
p.write_text(s.replace(old, "func ackDue(seen, current int) bool { return seen == 0 && current > 0 }"))
