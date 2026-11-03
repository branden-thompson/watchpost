import pathlib
# 0.19.3 D-360: the radio line is cut, not slid, under Reduce motion.
# Killed by TestReduceMotionCutsTheRadioLine.
p = pathlib.Path('modes/tty/radio_panel.go'); s = p.read_text()
old = "\t\tif d.reduceMotion {\n\t\t\ttext = cutToWidth("
assert s.count(old) == 1, 'm619'
p.write_text(s.replace(old, "\t\tif false {\n\t\t\ttext = cutToWidth(", 1))
