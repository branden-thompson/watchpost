import pathlib
# 0.19.0 D-134: the words state what the typical error was measured on.
# Killed by TestTheTypicalErrorStatesItsBasis.
p = pathlib.Path('modes/tty/map_prop_ahead.go'); s = p.read_text()
old = '\t\twords = append(words, b) // in words a listener can be told; the citation is the library\'s Source (AX-3)\n'
assert old in s, 'm212'
p.write_text(s.replace(old, '\t\t_ = b\n'))
