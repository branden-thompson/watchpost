import pathlib
# 0.19.3 D-360: the console carries the listener's Reduce motion.
# Killed by TestTheConsoleFollowsReduceMotion.
p = pathlib.Path('modes/tty/router.go'); s = p.read_text()
old = "\tif out.observer.reduceMotion { // AND REDUCE MOTION"
assert s.count(old) == 1, 'm622'
p.write_text(s.replace(old, "\tif false { // AND REDUCE MOTION", 1))
