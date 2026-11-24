import pathlib
# The console goes back to building five `DayCell`s per row for two tables that
# draw neither — 351 allocations a frame spent on nothing (D-120). The MEASURED
# half of that batch; the map half was measured and found not to be a win.
p = pathlib.Path("modes/tty/console/broadcaster_lineup.go"); s = p.read_text()
old = """(loc, b.fireBold(), 0)"""
assert old in s, "mAN1"
# Five is the root's extRowDays (modes/tty/body.go), which the console's package
# cannot name (0.19.6 B3).
p.write_text(s.replace(old, """(loc, b.fireBold(), 5)""", 1))
