import pathlib
# A complete report stops calling itself "Location Report, Full" and lists all
# four labels instead. The mock's own words go, and the running order spends a
# row's width saying what one phrase said — on the card the operator reads most.
p = pathlib.Path("platform/report/report.go"); s = p.read_text()
old = """	if s.Full() {
		return FullLabel
	}
"""
assert old in s, "mAW1"
p.write_text(s.replace(old, "", 1))
