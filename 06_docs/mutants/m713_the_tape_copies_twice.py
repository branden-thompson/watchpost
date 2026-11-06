import pathlib
# 0.19.5 Q2.12 (PF-6): a beat's bytes are pinned, not only its allocations: the
# tape's rune slice copied twice over is the same count and kilobytes more.
# Killed by TestBeatAllocBudget.
p = pathlib.Path('modes/tty/ticker.go'); s = p.read_text()
old = '\tloop := []rune(text + gap)\n'
assert s.count(old) == 1, 'm713'
p.write_text(s.replace(old, '\tloop := []rune(text + gap + text + gap)[:len([]rune(text + gap))]\n', 1))
