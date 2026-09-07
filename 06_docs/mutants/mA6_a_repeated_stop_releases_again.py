import pathlib
# The idempotence guard dropped, so a pump that delivers one command twice
# releases the band a second time.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = """	if d.power == ev.To {
		return d, nil // a repeated command is not a second event
	}
"""
assert old in s, "mA6"
p.write_text(s.replace(old, "", 1))
