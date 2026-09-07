import pathlib
# The schedule's stop neither cancels nor waits, so the pump loop, its lane and
# the tick goroutine outlive the station. A sender left writing to a loop that
# has returned is the shape that turned a 0.12.0 release tag red on the Linux
# race gate — green on macOS, caught by ubuntu CI.
p = pathlib.Path("app/schedule.go"); s = p.read_text()
old = """	s.cancel()
	<-s.ticks
	s.pump.stop()"""
new = """	_ = s.cancel"""
assert old in s, "mI0"
p.write_text(s.replace(old, new, 1))
