import pathlib
# 0.19.3 D-360: space plays no forecast under Reduce motion.
# Killed by TestReduceMotionStillsTheMap.
p = pathlib.Path('modes/tty/map_temp.go'); s = p.read_text()
old = "\t\tif d.reduceMotion { // nothing plays under Reduce motion; the timeline says so (D-360)\n"
assert s.count(old) == 1, 'm614'
p.write_text(s.replace(old, "\t\tif false { // nothing plays under Reduce motion; the timeline says so (D-360)\n", 1))
