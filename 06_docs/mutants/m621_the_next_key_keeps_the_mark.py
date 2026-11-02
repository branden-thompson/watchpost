import pathlib
# 0.19.3 D-360 (A-107 (10)): the next key clears the steady mark.
# Killed by TestReduceMotionMakesTheBlinksSteady.
p = pathlib.Path('modes/tty/motion.go'); s = p.read_text()
old = "\tif !d.reduceMotion {\n\t\treturn d\n\t}\n\td.volFlash"
assert s.count(old) == 1, 'm621'
p.write_text(s.replace(old, "\tif true {\n\t\treturn d\n\t}\n\td.volFlash", 1))
