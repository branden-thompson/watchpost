import pathlib
# 0.19.5 Q2.12 (PF-6): a beat's bytes are pinned, not only its allocations: the
# marquee's window reserving a lane's worth of kilobytes is no more allocations
# and many more bytes.
# Killed by TestBeatAllocBudget.
p = pathlib.Path('modes/tty/ticker.go'); s = p.read_text()
old = '\tvar b strings.Builder\n\tcells := 0\n\tfor i := 0; i < n && cells < width; i++ {\n\t\tr := tape.next()\n'
assert s.count(old) == 1, 'm713'
p.write_text(s.replace(old, '\tvar b strings.Builder\n\tb.Grow(64 << 10)\n\tcells := 0\n\tfor i := 0; i < n && cells < width; i++ {\n\t\tr := tape.next()\n', 1))
