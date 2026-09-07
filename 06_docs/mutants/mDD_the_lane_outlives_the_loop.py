import pathlib
# The lane is left running when the loop returns, so cancelling the context
# retires the pump but leaks the lane goroutine until someone calls stop.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = "\tdefer close(p.lane)\n"
assert old in s, "mDD"
p.write_text(s.replace(old, "", 1))
