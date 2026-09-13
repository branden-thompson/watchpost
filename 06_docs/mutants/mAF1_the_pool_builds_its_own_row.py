import pathlib
# The pool goes back to converting a snapshot into a row by hand, copying six of
# sixteen fields: no HI, no LOW, no TOMORROW at all, no trend arrow, no fire or
# seismic marks. Every one of those columns draws "n/a" on the table whose whole
# purpose is "basic weather info to determine if they want to have that location
# prioritized" (D-112).
p = pathlib.Path("modes/tty/broadcaster_pool.go"); s = p.read_text()
old = """			row = weatherRow(loc, bcFireBoldMW, 0)"""
assert old in s, "mAF1"
new = """			row.Loading = rowLoading(loc)
			row.Conditions = loc.Harmonized.Condition
			row.Now = loc.Harmonized.Temp
			row.Station = loc.Harmonized.Source.ModelOrStation"""
p.write_text(s.replace(old, new, 1))
