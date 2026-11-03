import pathlib
# 0.19.3 D-360: the loading dots hold still under Reduce motion.
# Killed by TestReduceMotionStillsTheVisualiserAndTheDots.
p = pathlib.Path('modes/tty/motion.go'); s = p.read_text()
old = "\tif d.reduceMotion {\n\t\treturn d\n\t}\n\td.frame++"
assert s.count(old) == 1, 'm618'
p.write_text(s.replace(old, "\tif false {\n\t\treturn d\n\t}\n\td.frame++", 1))
