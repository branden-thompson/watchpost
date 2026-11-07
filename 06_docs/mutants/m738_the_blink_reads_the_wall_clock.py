import pathlib
# 0.19.5 Q3.1: a blink is set on the model's clock, so a pinned clock pins it.
# Killed by TestABlinkEndsOnTheModelsClock.
p = pathlib.Path('modes/tty/motion.go'); s = p.read_text()
old = '\treturn d.clock().Add(pickerFlashDur)\n'
assert s.count(old) == 1, 'm738'
p.write_text(s.replace(old, '\treturn time.Now().Add(pickerFlashDur)\n', 1))
