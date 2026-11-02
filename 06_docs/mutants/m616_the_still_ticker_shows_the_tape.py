import pathlib
# 0.19.3 D-360 (A-107 (10)): the still ticker shows an item at a time, whole.
# Killed by TestReduceMotionStepsTheTicker.
p = pathlib.Path('modes/tty/ticker.go'); s = p.read_text()
old = "\t\tif d.reduceMotion { // an item at a time, whole, not a scrolling tape (D-360)\n"
assert s.count(old) == 1, 'm616'
p.write_text(s.replace(old, "\t\tif false { // an item at a time, whole, not a scrolling tape (D-360)\n", 1))
