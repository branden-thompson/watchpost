import pathlib
# 0.19.3 D-360 (A-107 (10)): under Reduce motion a pressed chip's mark is steady.
# Killed by TestReduceMotionMakesTheBlinksSteady.
p = pathlib.Path('modes/tty/motion.go'); s = p.read_text()
old = "\tif d.reduceMotion {\n\t\treturn time.Now().Add(steadyMark)"
assert s.count(old) == 1, 'm620'
p.write_text(s.replace(old, "\tif false {\n\t\treturn time.Now().Add(steadyMark)", 1))
