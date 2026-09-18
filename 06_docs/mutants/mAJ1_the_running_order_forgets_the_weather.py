import pathlib
# The running order stops joining the location's weather, so CONDITIONS and NOW
# go blank on every row — the columns the HUM LEAD spent the correspondent's
# twenty-eight cells on, because that one read `N/A` (D-116).
p = pathlib.Path("modes/tty/broadcaster_lineup.go"); s = p.read_text()
old = """			row.Conditions, row.Now, row.Trend, row.Loading = w.Conditions, w.Now, w.Trend, w.Loading"""
assert old in s, "mAJ1"
p.write_text(s.replace(old, "", 1))
