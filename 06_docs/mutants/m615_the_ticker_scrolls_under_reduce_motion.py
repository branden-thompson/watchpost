import pathlib
# 0.19.3 D-360: the tape does not scroll under Reduce motion.
# Killed by TestReduceMotionStepsTheTicker.
p = pathlib.Path('modes/tty/ticker.go'); s = p.read_text()
old = "\tif len(d.ticker) == 0 || d.breaking != nil || d.reduceMotion {\n"
assert s.count(old) == 1, 'm615'
p.write_text(s.replace(old, "\tif len(d.ticker) == 0 || d.breaking != nil {\n", 1))
