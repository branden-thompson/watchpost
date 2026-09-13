import pathlib
# The console goes back to building five `DayCell`s per row for two tables that
# draw neither — 351 allocations a frame spent on nothing (D-120). The MEASURED
# half of that batch; the map half was measured and found not to be a win.
p = pathlib.Path("modes/tty/broadcaster_lineup.go"); s = p.read_text()
old = """			w := weatherRow(loc, bcFireBoldMW, 0)"""
assert old in s, "mAN1"
p.write_text(s.replace(old, """			w := weatherRow(loc, bcFireBoldMW, extRowDays)""", 1))
