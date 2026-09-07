import pathlib
# The takeover's headline drops "+ N more", so the operator's card names the
# lead alert and says nothing about the two behind it. The read is unchanged —
# which is exactly why this is dangerous: the panel understates the hazard and
# nothing about the audio contradicts it.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = """	if len(p) > 1 {
		headline = fmt.Sprintf("%s + %d more", headline, len(p)-1)
	}"""
new = """	if len(p) > 1_000_000 {
		headline = fmt.Sprintf("%s + %d more", headline, len(p)-1)
	}"""
assert old in s, "mT1"
p.write_text(s.replace(old, new, 1))
