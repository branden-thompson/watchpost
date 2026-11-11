import pathlib
# 0.19.5 Q3.1: the tick ends a blink on the model's clock.
# Killed by TestABlinkEndsOnTheModelsClock.
p = pathlib.Path('modes/tty/dashboard.go'); s = p.read_text()
old = 'if d.volFlash != "" && !d.clock().Before(d.volFlashEnd) {'
assert s.count(old) == 1, 'm739'
p.write_text(s.replace(old, 'if d.volFlash != "" && !time.Now().Before(d.volFlashEnd) {', 1))
