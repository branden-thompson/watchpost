import pathlib
# 0.19.3 D-360: the bars are not drawn under Reduce motion.
# Killed by TestReduceMotionStillsTheVisualiserAndTheDots.
p = pathlib.Path('modes/tty/radio_panel.go'); s = p.read_text()
old = "\tif !d.radioViz || d.cfg.Spectrum == nil || d.reduceMotion {"
assert s.count(old) == 1, 'm617'
p.write_text(s.replace(old, "\tif !d.radioViz || d.cfg.Spectrum == nil {", 1))
