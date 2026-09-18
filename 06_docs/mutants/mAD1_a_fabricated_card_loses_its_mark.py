import pathlib
# A fabricated event stops saying so. The mark used to ride the card's title ROW —
# its own non-truncatable column — and D-110 put the title in the RULE; drop it
# here and a test event looks exactly like a real one, which is the screenshot
# hazard FR-4.4 exists to prevent (D-55, D-110).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	if !c.Test {
		return title
	}
	if title == "" {
		return testEventMark
	}
	return testEventMark + " " + title"""
assert old in s, "mAD1"
p.write_text(s.replace(old, "	return title", 1))
