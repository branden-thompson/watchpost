import pathlib
# 0.19.3 D-360: Reduce motion turns the library's ReduceMotion on.
# Killed by TestReduceMotionStillsTheMap.
p = pathlib.Path('modes/tty/motion.go'); s = p.read_text()
old = "\ton := d.reduceMotion\n"
assert s.count(old) == 1, 'm613'
p.write_text(s.replace(old, "\ton := false\n", 1))
