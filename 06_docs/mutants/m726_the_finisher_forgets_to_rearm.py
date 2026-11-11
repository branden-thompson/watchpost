import pathlib
# 0.19.5 Q3.4 (PF-4): the one-pass finisher re-arms the frame's foreground after
# every inner reset, as TintKeeping did over the joined frame.
# Killed by TestTheConsolesFinisherIsByteIdentical.
p = pathlib.Path('modes/tty/broadcaster.go'); s = p.read_text()
old = '\t\tout.WriteString(row[:k])\n\t\tout.WriteString(base)\n'
assert s.count(old) == 1, 'm726'
p.write_text(s.replace(old, '\t\tout.WriteString(row[:k])\n\t\tout.WriteString(reset)\n', 1))
